// lintian-ssg, a static site generator for lintian tags explanations.
//
// Copyright (C) Nicolas Peugnet <nicolas@club1.fr>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:generate sh -c "sed s/{{.version}}/$(git describe --tags --always --dirty)/ version.go.tmpl > version.go"

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
	"strings"
	"sync"

	"github.com/n-peugnet/lintian-ssg/markdown"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

const (
	outDir     = "out"
	manualPath = "/usr/share/doc/lintian/lintian.html"
)

type Tag struct {
	Name           string   `json:"name"`
	Visibility     string   `json:"visibility"`
	Explanation    string   `json:"explanation"`
	SeeAlso        []string `json:"see_also"`
	RenamedFrom    []string `json:"renamed_from"`
	Experimental   bool     `json:"experimental"`
	LintianVersion string   `json:"lintian_version"`
}

type TmplParams struct {
	Root           string
	Version        string
	VersionLintian string
	TagList        []string
	TagDatalist    template.HTML
}

type TagTmplParams struct {
	Tag
	TmplParams
	ExplanationHTML template.HTML
	SeeAlsoHTML     []template.HTML
	RenamedFromStr  string
}

type File struct {
	name    string
	content io.Reader
}

var (
	//go:embed index.html.tmpl
	indexTmplStr string
	//go:embed tag.html.tmpl
	tagTmplStr string
	//go:embed main.css
	mainCSS []byte
	//go:embed openlogo-50.svg
	logoSVG []byte
	//go:embed favicon.ico
	faviconICO []byte

	version  = "v0.1.0"
	mdParser = goldmark.New(
		goldmark.WithParser(parser.NewParser(
			parser.WithBlockParsers(
				// adapted from parser.DefaultBlockParsers(), with headings removed
				util.Prioritized(parser.NewThematicBreakParser(), 200),
				util.Prioritized(parser.NewListParser(), 300),
				util.Prioritized(parser.NewListItemParser(), 400),
				util.Prioritized(markdown.NewAnyIndentCodeBlockParser(), 500),
				util.Prioritized(parser.NewFencedCodeBlockParser(), 700),
				util.Prioritized(parser.NewBlockquoteParser(), 800),
				util.Prioritized(parser.NewHTMLBlockParser(), 900),
				util.Prioritized(parser.NewParagraphParser(), 1000),
			),
			parser.WithInlineParsers(parser.DefaultInlineParsers()...),
			parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
		)),
		goldmark.WithRendererOptions(renderer.WithNodeRenderers(
			util.Prioritized(markdown.NewIndentedCodeBlockRenderer(), 500),
		)),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
)

func rootRelPath(dir string) string {
	count := strings.Count(dir, "/")
	return strings.Repeat("../", count+1)
}

func md2html(src string) (template.HTML, error) {
	buf := bytes.Buffer{}
	err := mdParser.Convert([]byte(src), &buf)
	return template.HTML(buf.String()), err
}

func buildTmplParams(tag *Tag, params TmplParams) *TagTmplParams {
	tmplParams := &TagTmplParams{
		Tag:            *tag,
		TmplParams:     params,
		RenamedFromStr: strings.Join(tag.RenamedFrom, ", "),
	}
	html, err := md2html(tag.Explanation)
	if err != nil {
		log.Println("WARNING: convert markdown explanation:", err)
	}
	tmplParams.ExplanationHTML = html
	tmplParams.SeeAlsoHTML = make([]template.HTML, len(tag.SeeAlso))
	for i, str := range tag.SeeAlso {
		html, err := md2html(str)
		if err != nil {
			log.Printf("WARNING: convert markdown reference %d: %v", i, err)
		}
		tmplParams.SeeAlsoHTML[i] = template.HTML(html)
	}
	return tmplParams
}

func renderTag(tag *Tag, params TmplParams, tmpl *template.Template, wg *sync.WaitGroup) {
	defer wg.Done()
	dir, name := path.Split(tag.Name)
	dirPath := filepath.Join(outDir, "tags", dir)
	filePath := filepath.Join(dirPath, name+".html")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		panic(err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	params.Root = rootRelPath(dir)
	if err := tmpl.Execute(file, buildTmplParams(tag, params)); err != nil {
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
		{"main.css", bytes.NewReader(mainCSS)},
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

func writeIndex(indexTmpl *template.Template, params TmplParams) error {
	file, err := os.Create(filepath.Join(outDir, "index.html"))
	if err != nil {
		return err
	}
	defer file.Close()
	params.Root = "./"
	return indexTmpl.Execute(file, params)
}

func main() {
	log.SetFlags(0)

	indexTmpl := template.Must(template.New("index").Parse(indexTmplStr))
	tagTmpl, err := template.Must(indexTmpl.Clone()).Parse(tagTmplStr)
	if err != nil {
		panic(err)
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
	tagTmpl.ExecuteTemplate(&buf, "lintian-tags", listTagsLines)
	tagDatalist := buf.String()
	params := TmplParams{
		Version:     version,
		TagList:     listTagsLines,
		TagDatalist: template.HTML(tagDatalist),
	}

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
		if params.VersionLintian == "" {
			params.VersionLintian = tag.LintianVersion
		}
		wg.Add(1)
		go renderTag(&tag, params, tagTmpl, &wg)
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
	if err := writeIndex(indexTmpl, params); err != nil {
		log.Fatalln("ERROR: write index.html:", err)
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
