// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package lintian

import (
	"fmt"
	"html/template"
	"path"
	"strings"

	"github.com/n-peugnet/lintian-ssg/markdown"
)

const sourceURLFmt = "https://salsa.debian.org/lintian/lintian/-/blob/%s/tags/%s.tag"

type Screen struct {
	Advocates []string `json:"advocates"`
	Name      string   `json:"name"`
	Reason    string   `json:"reason"`
	SeeAlso   []string `json:"see_also"`
}

func (s *Screen) AdvocatesHTML() template.HTML {
	return markdown.ToHTML(strings.Join(s.Advocates, ", "), markdown.StyleInline)
}

func (s *Screen) ReasonHTML() template.HTML {
	return markdown.ToHTML(s.Reason, markdown.StyleFull)
}

func (s *Screen) SeeAlsoHTML() template.HTML {
	return markdown.ToHTML("See also: "+strings.Join(s.SeeAlso, ", "), markdown.StyleInline)
}

type Level string

const (
	LevelError          Level = "error"
	LevelWarning        Level = "warning"
	LevelInfo           Level = "info"
	LevelPedantic       Level = "pedantic"
	LevelClassification Level = "classification"
)

type Tag struct {
	Name           string   `json:"name"`
	NameSpaced     bool     `json:"name_spaced"`
	Visibility     Level    `json:"visibility"`
	Explanation    string   `json:"explanation"`
	SeeAlso        []string `json:"see_also"`
	RenamedFrom    []string `json:"renamed_from"`
	Experimental   bool     `json:"experimental"`
	LintianVersion string   `json:"lintian_version"`
	Screens        []Screen `json:"screens"`
}

func (t *Tag) ExplanationHTML() template.HTML {
	return markdown.ToHTML(t.Explanation, markdown.StyleFull)
}

func (t *Tag) SeeAlsoHTML() []template.HTML {
	seeAlsoHTML := make([]template.HTML, len(t.SeeAlso))
	for i, str := range t.SeeAlso {
		seeAlsoHTML[i] = markdown.ToHTML(str, markdown.StyleInline)
	}
	return seeAlsoHTML
}

func (t *Tag) RenamedFromStr() string {
	return strings.Join(t.RenamedFrom, ", ")
}

func (t *Tag) Source() string {
	name := t.Name
	if !t.NameSpaced {
		name = path.Join(string(name[0]), name)
	}
	return fmt.Sprintf(sourceURLFmt, t.LintianVersion, name)
}
