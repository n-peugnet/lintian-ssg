// Copyright (C) Nicolas Peugnet <nicolas@club1.fr>

package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	outDir = "out"
)

type Tag struct {
	Name         string   `json:"name"`
	Visibility   string   `json:"visibility"`
	Explanation  string   `json:"explanation"`
	SeeAlso      []string `json:"see_also"`
	RenamedFrom  []string `json:"renamed_from"`
	Experimental bool     `json:"experimental"`
}

type TmplParams struct {
	Tag
	ExplanationHTML template.HTML
	SeeAlsoHTML     []template.HTML
	RenamedFromStr  string
}

var (
	//go:embed tag.html.tmpl
	tagTmpl    string
	linkRegex1 = regexp.MustCompile(`<(\S+)>`)
	linkRegex2 = regexp.MustCompile(`\[([^]]+)\]\((\S+)\)`)
)

func convertLinks(str string) string {
	str = linkRegex1.ReplaceAllString(str, `<a href="${1}">${1}</a>`)
	str = linkRegex2.ReplaceAllString(str, `<a href="${2}">${1}</a>`)
	return str
}

func tag2tmplParams(tag *Tag) *TmplParams {
	tmplTag := &TmplParams{
		Tag:             *tag,
		ExplanationHTML: template.HTML(tag.Explanation),
		RenamedFromStr:  strings.Join(tag.RenamedFrom, ", "),
	}
	tmplTag.SeeAlsoHTML = make([]template.HTML, len(tag.SeeAlso))
	for i, str := range tag.SeeAlso {
		tmplTag.SeeAlsoHTML[i] = template.HTML(convertLinks(str))
	}
	return tmplTag
}

func renderTag(tag *Tag, tmpl *template.Template, wg *sync.WaitGroup) {
	defer wg.Done()
	dir, name := path.Split(tag.Name)
	dirPath := filepath.Join(outDir, dir)
	filePath := filepath.Join(dirPath, name+".html")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		panic(err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	if err := tmpl.Execute(file, tag2tmplParams(tag)); err != nil {
		panic(err)
	}
}

func main() {
	log.SetFlags(0)

	listTagsOut := strings.Builder{}
	listTagsCmd := exec.Command("lintian-explain-tags", "--list-tags")
	listTagsCmd.Stderr = os.Stderr
	listTagsCmd.Stdout = &listTagsOut
	if err := listTagsCmd.Run(); err != nil {
		log.Fatalln("ERROR:", err)
	}
	listTagsStr := listTagsOut.String()
	listTagsLines := strings.Split(listTagsStr, "\n")

	jsonTagsArgs := append([]string{"--format=json"}, listTagsLines...)
	jsonTagsCmd := exec.Command("lintian-explain-tags", jsonTagsArgs...)
	jsonTagsCmd.Stderr = os.Stderr
	jsonTagsOut, err := jsonTagsCmd.StdoutPipe()
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
	jsonTagsDecoder := json.NewDecoder(jsonTagsOut)
	if err := jsonTagsCmd.Start(); err != nil {
		log.Fatalln("ERROR", err)
	}

	var tags []Tag
	if err := jsonTagsDecoder.Decode(&tags); err != nil {
		log.Fatalln("ERROR:", err)
	}

	tmpl, err := template.New(tagTmpl).Parse(tagTmpl)
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
	wg := sync.WaitGroup{}
	wg.Add(len(tags))
	for i := range tags {
		go renderTag(&tags[i], tmpl, &wg)
	}
	wg.Wait()

	if err := jsonTagsCmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			log.Println("WARNING: lintian-explain-tags returned non zero exit status:", exitError.ExitCode())
		} else {
			log.Println("ERROR: running lintian-explain-tags:", err)
		}
	}
}
