// Copyright (C) Nicolas Peugnet <nicolas@club1.fr>

package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"io"
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
	outDir     = "out"
	manualPath = "/usr/share/doc/lintian/lintian.html"
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
	TagDatalist     template.HTML
	ExplanationHTML template.HTML
	SeeAlsoHTML     []template.HTML
	RenamedFromStr  string
	Root            string
}

type File struct {
	name    string
	content io.Reader
}

var (
	//go:embed tag.html.tmpl
	tagTmpl string
	//go:embed tag.css
	tagCSS []byte
	//go:embed openlogo-50.svg
	logoSVG []byte
	//go:embed favicon.ico
	faviconICO []byte

	linkRegex1 = regexp.MustCompile(`<(\S+)>`)
	linkRegex2 = regexp.MustCompile(`\[([^]]+)\]\((\S+)\)`)
)

func rootRelPath(dir string) string {
	count := strings.Count(dir, "/")
	if count == 0 {
		return "./"
	}
	return strings.Repeat("../", count)
}

func convertLinks(str string) string {
	str = linkRegex1.ReplaceAllString(str, `<a href="${1}">${1}</a>`)
	str = linkRegex2.ReplaceAllString(str, `<a href="${2}">${1}</a>`)
	return str
}

func buildTmplParams(tag *Tag, tagDatalist string, dir string) *TmplParams {
	tmplTag := &TmplParams{
		Tag:             *tag,
		TagDatalist:     template.HTML(tagDatalist),
		ExplanationHTML: template.HTML(tag.Explanation),
		RenamedFromStr:  strings.Join(tag.RenamedFrom, ", "),
		Root:            rootRelPath(dir),
	}
	tmplTag.SeeAlsoHTML = make([]template.HTML, len(tag.SeeAlso))
	for i, str := range tag.SeeAlso {
		tmplTag.SeeAlsoHTML[i] = template.HTML(convertLinks(str))
	}
	return tmplTag
}

func renderTag(tag *Tag, tags string, tmpl *template.Template, wg *sync.WaitGroup) {
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
	if err := tmpl.ExecuteTemplate(file, "main", buildTmplParams(tag, tags, dir)); err != nil {
		panic(err)
	}
}

func writeFiles(files []File) error {
	for _, f := range files {
		path := filepath.Join(outDir, f.name)
		dir, _ := filepath.Split(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := io.Copy(file, f.content); err != nil {
			return err
		}
	}
	return nil
}

func writeAssets() error {
	return writeFiles([]File{
		{"tag.css", bytes.NewReader(tagCSS)},
		{"openlogo-50.svg", bytes.NewReader(logoSVG)},
		{"favicon.ico", bytes.NewReader(faviconICO)},
	})
}

func writeManual() error {
	file, err := os.Open(manualPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeFiles([]File{{"manual/index.html", file}})
}

func main() {
	log.SetFlags(0)

	tmpl, err := template.New(tagTmpl).Parse(tagTmpl)
	if err != nil {
		log.Fatalln("ERROR:", err)
	}

	listTagsOut := strings.Builder{}
	listTagsCmd := exec.Command("lintian-explain-tags", "--list-tags")
	listTagsCmd.Stderr = os.Stderr
	listTagsCmd.Stdout = &listTagsOut
	if err := listTagsCmd.Run(); err != nil {
		log.Fatalln("ERROR:", err)
	}
	listTagsStr := strings.TrimSpace(listTagsOut.String())
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
		log.Fatalln("ERROR:", err)
	}

	buf := bytes.Buffer{}
	tmpl.ExecuteTemplate(&buf, "lintian-tags", listTagsLines)
	tagDatalist := buf.String()

	// discard open bracket
	if _, err := jsonTagsDecoder.Token(); err != nil {
		log.Fatalln("ERROR:", err)
	}

	wg := sync.WaitGroup{}
	// while the array contains values
	for jsonTagsDecoder.More() {
		var tag Tag
		if err := jsonTagsDecoder.Decode(&tag); err != nil {
			log.Fatalln("ERROR:", err)
		}
		wg.Add(1)
		go renderTag(&tag, tagDatalist, tmpl, &wg)
	}

	// discard closing bracket
	if _, err = jsonTagsDecoder.Token(); err != nil {
		log.Fatalln("ERROR:", err)
	}

	if err := writeAssets(); err != nil {
		log.Fatalln("ERROR: write assets:", err)
	}
	if err := writeManual(); err != nil {
		log.Fatalln("ERROR: write manual:", err)
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
