// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package markdown_test

import (
	"fmt"
	"testing"

	"github.com/n-peugnet/lintian-ssg/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/testutil"
	"github.com/yuin/goldmark/util"
)

func TestManpageLink(t *testing.T) {
	markdown := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithParserOptions(parser.WithInlineParsers(
			util.Prioritized(markdown.NewManpageLinkParser(), 500),
		)),
	)
	cases := []struct {
		src      string
		expected string
	}{
		{ // basic case
			`see lintian(1).`,
			`<p>see <a href="https://manpages.debian.org/lintian(1)">lintian(1)</a>.</p>`,
		},
		{ // with special chars
			`update-rc.d(8)`,
			`<p><a href="https://manpages.debian.org/update-rc.d(8)">update-rc.d(8)</a></p>`,
		},
		{ // inside <code></code>
			`see <code>lintian(1)</code>.`,
			`<p>see <code><a href="https://manpages.debian.org/lintian(1)">lintian(1)</a></code>.</p>`,
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case %d %s", i, c.src), func(t *testing.T) {
			testutil.DoTestCase(
				markdown,
				testutil.MarkdownTestCase{
					No:       i,
					Markdown: c.src,
					Expected: c.expected,
				},
				t,
			)
		})
	}
}
