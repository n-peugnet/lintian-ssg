// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package goldmark_ext

import (
	"fmt"
	"regexp"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// manpageURLFmt is the format to use when producing the URL of a manpage.
const manpageURLFmt = "https://manpages.debian.org/%s"

var manpageLinkRegexp = regexp.MustCompile(`^[-\w\.]+\([1-9]\)`)

type manpageLinkParser struct{}

// NewManpageLinkParser returns a new InlineParser that parses manpage links
// in the form pagename(n).
func NewManpageLinkParser() parser.InlineParser {
	return &manpageLinkParser{}
}

func (p *manpageLinkParser) Trigger() []byte {
	// ' ' indicates any white spaces and a line head
	return []byte{' '}
}

func (p *manpageLinkParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	if pc.IsInLinkLabel() {
		return nil
	}
	line, segment := block.PeekLine()
	consumes := 0
	start := segment.Start
	if line[0] == ' ' {
		line = line[1:]
		consumes++
		start++
	}
	loc := manpageLinkRegexp.FindIndex(line)
	if loc == nil {
		return nil
	}

	// Create new node
	stop := loc[1]
	match := string(line[loc[0]:stop])
	text := ast.NewTextSegment(text.NewSegment(start, start+stop))
	buf := make([]byte, 0, len(manpageURLFmt)+10)
	url := fmt.Appendf(buf, manpageURLFmt, match)
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
