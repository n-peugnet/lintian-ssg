// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package markdown_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/n-peugnet/lintian-ssg/markdown"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/testutil"
)

type dummyGoldmark struct {
	style markdown.Style
}

func (g *dummyGoldmark) Convert(source []byte, writer io.Writer, opts ...parser.ParseOption) error {
	html := markdown.ToHTML(string(source), "test", g.style)
	if _, err := writer.Write([]byte(html)); err != nil {
		return err
	}
	return nil
}

// Dummy implementations
func (g *dummyGoldmark) Parser() parser.Parser         { return nil }
func (g *dummyGoldmark) SetParser(parser.Parser)       {}
func (g *dummyGoldmark) Renderer() renderer.Renderer   { return nil }
func (g *dummyGoldmark) SetRenderer(renderer.Renderer) {}

func TestDataInline(t *testing.T) {
	cases := []struct {
		src  string
		html string
	}{
		{ // Headers are not parsed
			"# no header",
			"<p># no header</p>",
		},
	}
	md := &dummyGoldmark{markdown.StyleInline}
	for i, c := range cases {
		name := fmt.Sprintf("%d %s", i, c.src)
		t.Run(name, func(t *testing.T) {
			testutil.DoTestCase(md, testutil.MarkdownTestCase{
				No:       i,
				Markdown: c.src,
				Expected: c.html,
			}, t)
		})
	}
}

func TestDataFull(t *testing.T) {
	srcs, err := filepath.Glob("testdata/*.full.md")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	md := &dummyGoldmark{markdown.StyleFull}
	for i, src := range srcs {
		markdown, err := os.ReadFile(src)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		expectedFile := src[:len(src)-3] + ".html"
		expected, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Fatal("unexpected error:", err)
		}
		t.Run(src, func(t *testing.T) {
			testutil.DoTestCase(md, testutil.MarkdownTestCase{
				No:       i,
				Markdown: string(markdown),
				Expected: string(expected),
			}, t)
		})
	}
}
