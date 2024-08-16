// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package goldmark_ext_test

import (
	"fmt"
	"testing"

	"github.com/n-peugnet/lintian-ssg/markdown/goldmark_ext"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/testutil"
	"github.com/yuin/goldmark/util"
)

func TestBugLink(t *testing.T) {
	markdown := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithParserOptions(parser.WithInlineParsers(
			util.Prioritized(goldmark_ext.NewBugLinkParser(), 500),
		)),
	)
	cases := []struct {
		src      string
		expected string
	}{
		{ // basic case
			`see Bug#12345.`,
			`<p>see <a href="https://bugs.debian.org/12345">Bug#12345</a>.</p>`,
		},
		{ // In parenthesis
			`(Bug#12345)`,
			`<p>(<a href="https://bugs.debian.org/12345">Bug#12345</a>)</p>`,
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
