// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package goldmark_ext

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// bugURLFmt is the format to use when producing the URL of a bug.
const bugURLFmt = "https://bugs.debian.org/%d"

var bugLinkRegexp = regexp.MustCompile(`^Bug#(\d+)\b`)

type bugLinkParser struct{}

// NewBugLinkParser returns a new InlineParser that parses bug links
// in the form Bug#nnnnn .
func NewBugLinkParser() parser.InlineParser {
	return &bugLinkParser{}
}

func (p *bugLinkParser) Trigger() []byte {
	// ' ' indicates any white spaces and a line head
	return []byte{' ', '('}
}

func (p *bugLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	if pc.IsInLinkLabel() {
		return nil
	}
	line, segment := block.PeekLine()
	consumes := 0
	start := segment.Start
	switch line[0] {
	case ' ', '(':
		line = line[1:]
		consumes++
		start++
	}
	loc := bugLinkRegexp.FindSubmatchIndex(line)
	if loc == nil {
		return nil
	}

	// Create new node
	stop := loc[3]
	match := string(line[loc[2]:stop])
	number, _ := strconv.Atoi(match)
	text := ast.NewTextSegment(text.NewSegment(start, start+stop))
	buf := make([]byte, 0, len(bugURLFmt)+10)
	url := fmt.Appendf(buf, bugURLFmt, number)
	node := ast.NewLink()
	node.Destination = url
	node.AppendChild(node, text)

	// Adjust parser state
	block.Advance(stop + consumes)
	if consumes != 0 {
		s := segment.WithStop(segment.Start + consumes)
		ast.MergeOrAppendTextSegment(parent, s)
	}
	return node
}
