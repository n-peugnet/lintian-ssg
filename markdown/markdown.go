// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package markdown

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/n-peugnet/lintian-ssg/markdown/goldmark_ext"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var (
	mdInline = goldmark.New(goldmark.WithParser(parser.NewParser(
		parser.WithBlockParsers(util.Prioritized(parser.NewParagraphParser(), 100)),
		parser.WithInlineParsers(parser.DefaultInlineParsers()...),
	)))
	mdFull = goldmark.New(
		goldmark.WithParser(parser.NewParser(
			parser.WithBlockParsers(
				// adapted from parser.DefaultBlockParsers(), with headings removed
				util.Prioritized(parser.NewThematicBreakParser(), 200),
				util.Prioritized(parser.NewListParser(), 300),
				util.Prioritized(parser.NewListItemParser(), 400),
				util.Prioritized(goldmark_ext.NewAnyIndentCodeBlockParser(), 500),
				util.Prioritized(parser.NewFencedCodeBlockParser(), 700),
				util.Prioritized(parser.NewBlockquoteParser(), 800),
				util.Prioritized(parser.NewHTMLBlockParser(), 900),
				util.Prioritized(parser.NewParagraphParser(), 1000),
			),
			parser.WithInlineParsers(append(
				parser.DefaultInlineParsers(),
				util.Prioritized(goldmark_ext.NewManpageLinkParser(), 1000),
				util.Prioritized(goldmark_ext.NewBugLinkParser(), 1000),
			)...),
			parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
		)),
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithExtensions(extension.Linkify),
	)
	// htmlEntReplacer is a strings.Replacer that transform some HTML entities
	// into their unicode representation.
	htmlEntReplacer = strings.NewReplacer(
		"&lowbar;", "_",
		"&lt;", "<",
		"&gt;", ">",
		"&ast;", "*",
	)
)

type Style int

const (
	StyleInline Style = iota
	StyleFull
)

func ToHTML(src string, style Style) template.HTML {
	var err error
	buf := bytes.Buffer{}
	switch style {
	case StyleInline:
		err = mdInline.Convert([]byte(src), &buf)
	case StyleFull:
		// Lintian tags explanation have had their underscores (_) replaced by
		// &lowbar; in lintian#d590cbf22, as well as some other special chars,
		// to fix plain text CLI output. Unfortunately it causes problems when
		// rendering markdown code blocks, so we simply replace them back, as
		// they will be escaped as needed by goldmark.
		src = htmlEntReplacer.Replace(src)
		err = mdFull.Convert([]byte(src), &buf)
	}
	if err != nil {
		// As we use a bytes.Buffer, goldmark.Convert should never return errors.
		panic(err)
	}
	return template.HTML(buf.String())
}
