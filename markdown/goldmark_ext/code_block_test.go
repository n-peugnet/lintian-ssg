// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package goldmark_ext_test

import (
	"fmt"
	"testing"

	"github.com/n-peugnet/lintian-ssg/markdown/goldmark_ext"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/testutil"
	"github.com/yuin/goldmark/util"
)

func TestAnyIndentCodeBlock(t *testing.T) {
	markdown := goldmark.New(goldmark.WithParser(parser.NewParser(parser.WithBlockParsers(
		util.Prioritized(goldmark_ext.NewAnyIndentCodeBlockParser(), 500),
		util.Prioritized(parser.NewParagraphParser(), 1000),
	))))
	cases := []struct {
		name     string
		src      string
		expected string
	}{
		{
			"indent of one space",
			`
 code
 block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{
			"indent of two space",
			`
  code
  block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{
			"indent of three spaces",
			`
   code
   block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{
			"indent of four spaces",
			`
    code
    block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{
			"indent of one tab",
			`
	code
	block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{
			"second line more indented",
			`
  code
      block
`,
			"<pre><code>code\n    block\n</code></pre>",
		},
		{
			"first line more indented",
			`
        code
    block
`,
			"<pre><code>    code\nblock\n</code></pre>",
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case %d %s", i, c.name), func(t *testing.T) {
			testutil.DoTestCase(
				markdown,
				testutil.MarkdownTestCase{
					No:          i,
					Description: c.name,
					Markdown:    c.src,
					Expected:    c.expected,
				},
				t,
			)
		})
	}
}
