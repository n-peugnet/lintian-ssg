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

func TestManpageLink(t *testing.T) {
	markdown := goldmark.New(
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
		goldmark.WithParserOptions(parser.WithInlineParsers(
			util.Prioritized(goldmark_ext.NewManpageLinkParser(), 500),
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
		{ // In link label
			`[lintian(1)](http://another.url)`,
			`<p><a href="http://another.url">lintian(1)</a></p>`,
		},
		{ // In link label with leading space
			`[manual page lintian(1)](http://another.url)`,
			`<p><a href="http://another.url">manual page lintian(1)</a></p>`,
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("%d %s", i, c.src), func(t *testing.T) {
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
