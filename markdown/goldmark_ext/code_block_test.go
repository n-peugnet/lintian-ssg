// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: MIT

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
		util.Prioritized(parser.NewListParser(), 300),
		util.Prioritized(parser.NewListItemParser(), 400),
		util.Prioritized(goldmark_ext.NewAnyIndentCodeBlockParser(), 500),
		util.Prioritized(parser.NewParagraphParser(), 1000),
	))))
	cases := []struct {
		name     string
		src      string
		expected string
	}{
		{"indent of one space", `
 code
 block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{"indent of two space", `
  code
  block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{"indent of three spaces", `
   code
   block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{"indent of four spaces", `
    code
    block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{"indent of one tab", `
	code
	block
`,
			"<pre><code>code\nblock\n</code></pre>",
		},
		{"indent of one tab leading tab", `
		code
	block
`,
			"<pre><code>	code\nblock\n</code></pre>",
		},
		{"second line more indented", `
  code
      block
`,
			"<pre><code>code\n    block\n</code></pre>",
		},
		{"first line more indented", `
        code
    block
`,
			"<pre><code>    code\nblock\n</code></pre>",
		},
		{"empty line in code block", `
	text

	code
	block
`,
			"<pre><code>text\n\ncode\nblock\n</code></pre>",
		},
		{"leading text tab indent", `
text

	code
	block
`,
			"<p>text</p>\n<pre><code>code\nblock\n</code></pre>",
		},
		{"cannot interrupt paragraph", `
text
	code
	block
`,
			"<p>text\ncode\nblock</p>",
		},
		{"inside list item one space", `
- item:

   code
   block
`,
			"<ul>\n<li>\n<p>item:</p>\n<pre><code>code\nblock\n</code></pre>\n</li>\n</ul>",
		},
		{"inside list item two spaces", `
- item:

    code
    block
`,
			"<ul>\n<li>\n<p>item:</p>\n<pre><code>code\nblock\n</code></pre>\n</li>\n</ul>",
		},
		{"inside list item four spaces leading tab", `
- item:

      	code
      block
`,
			"<ul>\n<li>\n<p>item:</p>\n<pre><code>	code\nblock\n</code></pre>\n</li>\n</ul>",
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("%d %s", i, c.name), func(t *testing.T) {
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
