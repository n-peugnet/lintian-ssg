// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// IndentedCodeBlockRenderer is a renderer.NodeRenderer implementation that
// renders code blocks differently besed on their indentation level.
type IndentedCodeBlockRenderer struct {
	html.Config
}

// NewIndentedCodeBlockRenderer returns a new IndentedCodeBlockRenderer.
func NewIndentedCodeBlockRenderer() *IndentedCodeBlockRenderer {
	return &IndentedCodeBlockRenderer{html.NewConfig()}
}

// RegisterFuncs implements renderer.NodeRenderer.RegisterFuncs.
func (r *IndentedCodeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindIndentedCodeBlock, r.renderAnyIndentCodeBlock)
}

func (r *IndentedCodeBlockRenderer) renderAnyIndentCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<pre><code>")
		r.writeLines(w, source, n)
	} else {
		_, _ = w.WriteString("</code></pre>\n")
	}
	return ast.WalkContinue, nil
}

func (r *IndentedCodeBlockRenderer) writeLines(w util.BufWriter, source []byte, n ast.Node) {
	node := n.(*IndentedCodeBlock)
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		if node.Indent == indentMax {
			// escape HTML special chars to make sure they are rendered as in the source
			r.Writer.RawWrite(w, line.Value(source))
		} else {
			// write codeblock's text as is in the HTML output
			r.Writer.Write(w, line.Value(source))
		}
	}
}
