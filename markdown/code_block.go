// This file is mainly a copy of github.com/yuin/goldmark/parser/code_block.go
//
// SPDX-FileCopyrightText: 2019 Yusuke Inuzuka
// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: MIT

package markdown

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const (
	indentMin = 1
	indentMax = 4
)

// KindIndentedCodeBlock is the NodeKind of the IndentedCodeBlock node.
var KindIndentedCodeBlock = ast.NewNodeKind("IndentedCodeBlock")

// IndentedCodeBlock represents an indented code block of Markdown text.
type IndentedCodeBlock struct {
	ast.CodeBlock
	Indent int
}

// NewIndentedCodeBlock returns a new CodeBlock node.
func NewIndentedCodeBlock(indent int) *IndentedCodeBlock {
	return &IndentedCodeBlock{
		CodeBlock: ast.CodeBlock{BaseBlock: ast.BaseBlock{}},
		Indent:    indent,
	}
}

// Kind implements Node.Kind.
func (n *IndentedCodeBlock) Kind() ast.NodeKind {
	return KindIndentedCodeBlock
}

type anyIndentCodeBlockParser struct {
	currentIndent int
}

// NewAnyIndentCodeBlockParser returns a new BlockParser that
// parses code blocks indented with any number of space (between 1-4) or tab.
func NewAnyIndentCodeBlockParser() parser.BlockParser {
	return &anyIndentCodeBlockParser{}
}

func (b *anyIndentCodeBlockParser) Trigger() []byte {
	return nil
}

func (b *anyIndentCodeBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, segment := reader.PeekLine()
	var i, pos, padding int
	for i = indentMin; i <= indentMax; i++ {
		pos, padding = util.IndentPosition(line, reader.LineOffset(), i)
		if pos >= 0 && !util.IsBlank(line) {
			break
		}
	}
	if i > indentMax {
		return nil, parser.NoChildren
	}
	b.currentIndent = i
	node := NewIndentedCodeBlock(i)
	reader.AdvanceAndSetPadding(pos, padding)
	_, segment = reader.PeekLine()
	// if code block line starts with a tab, keep a tab as it is.
	if segment.Padding != 0 {
		preserveLeadingTabInCodeBlock(&segment, reader, 0)
	}
	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return node, parser.NoChildren

}

func (b *anyIndentCodeBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if util.IsBlank(line) {
		node.Lines().Append(segment.TrimLeftSpaceWidth(b.currentIndent, reader.Source()))
		return parser.Continue | parser.NoChildren
	}
	pos, padding := util.IndentPosition(line, reader.LineOffset(), b.currentIndent)
	if pos < 0 {
		return parser.Close
	}
	reader.AdvanceAndSetPadding(pos, padding)
	_, segment = reader.PeekLine()

	// if code block line starts with a tab, keep a tab as it is.
	if segment.Padding != 0 {
		preserveLeadingTabInCodeBlock(&segment, reader, 0)
	}

	node.Lines().Append(segment)
	reader.Advance(segment.Len() - 1)
	return parser.Continue | parser.NoChildren
}

func (b *anyIndentCodeBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// trim trailing blank lines
	lines := node.Lines()
	length := lines.Len() - 1
	source := reader.Source()
	for length >= 0 {
		line := lines.At(length)
		if util.IsBlank(line.Value(source)) {
			length--
		} else {
			break
		}
	}
	lines.SetSliced(0, length+1)
}

func (b *anyIndentCodeBlockParser) CanInterruptParagraph() bool {
	return false
}

func (b *anyIndentCodeBlockParser) CanAcceptIndentedLine() bool {
	return true
}

func preserveLeadingTabInCodeBlock(segment *text.Segment, reader text.Reader, indent int) {
	offsetWithPadding := reader.LineOffset() + indent
	sl, ss := reader.Position()
	reader.SetPosition(sl, text.NewSegment(ss.Start-1, ss.Stop))
	if offsetWithPadding == reader.LineOffset() {
		segment.Padding = 0
		segment.Start--
	}
	reader.SetPosition(sl, ss)
}
