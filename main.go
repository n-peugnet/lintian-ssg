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
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/n-peugnet/lintian-ssg/ioutil"
	"github.com/n-peugnet/lintian-ssg/markdown"
)

const (
	outDir       = "out"
	manualPath   = "/usr/share/doc/lintian/lintian.html"
	sourceURLFmt = "https://salsa.debian.org/lintian/lintian/-/blob/%s/tags/%s.tag"
)

type Screen struct {
	Advocates []string `json:"advocates"`
	Name      string   `json:"name"`
	Reason    string   `json:"reason"`
	SeeAlso   []string `json:"see_also"`
}

func (s *Screen) AdvocatesHTML() template.HTML {
	return markdown.ToHTML(strings.Join(s.Advocates, ", "), "screen advocates", markdown.StyleInline)
}

func (s *Screen) ReasonHTML() template.HTML {
	return markdown.ToHTML(s.Reason, "screen reason", markdown.StyleFull)
}

func (s *Screen) SeeAlsoHTML() template.HTML {
	return markdown.ToHTML("See also: "+strings.Join(s.SeeAlso, ", "), "screen see_also", markdown.StyleInline)
}

type Tag struct {
	Name           string   `json:"name"`
	NameSpaced     bool     `json:"name_spaced"`
	Visibility     string   `json:"visibility"`
	Explanation    string   `json:"explanation"`
	SeeAlso        []string `json:"see_also"`
	RenamedFrom    []string `json:"renamed_from"`
	Experimental   bool     `json:"experimental"`
	LintianVersion string   `json:"lintian_version"`
	Screens        []Screen `json:"screens"`
}

func (t *Tag) ExplanationHTML() template.HTML {
	return markdown.ToHTML(t.Explanation, "explanation", markdown.StyleFull)
}

func (t *Tag) SeeAlsoHTML() []template.HTML {
	seeAlsoHTML := make([]template.HTML, len(t.SeeAlso))
	for i, str := range t.SeeAlso {
		seeAlsoHTML[i] = markdown.ToHTML(str, fmt.Sprintf("reference %d", i), markdown.StyleInline)
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

type TmplParams struct {
	DateYear       int
	DateHuman      string
	DateMachine    string
	BaseURL        string
	Root           string
	Version        string
	VersionLintian string
	TagList        []string
}

type ManualTmplParams struct {
	TmplParams
	Manual template.HTML
}

type TagTmplParams struct {
	TmplParams
	*Tag
	PrevName string
}

type File struct {
	name    string
	content io.Reader
}

var (
	//go:embed templates/index.html.tmpl
	indexTmplStr string
	//go:embed templates/tag.html.tmpl
	tagTmplStr string
	//go:embed templates/renamed.html.tmpl
	renamedTmplStr string
	//go:embed templates/manual.html.tmpl
	manualTmplStr string
	//go:embed templates/about.html.tmpl
	aboutTmplStr string
	//go:embed templates/404.html.tmpl
	e404TmplStr string
	//go:embed assets/main.css
	mainCSS []byte
	//go:embed assets/openlogo-50.svg
	logoSVG []byte
	//go:embed assets/favicon.ico
	faviconICO []byte

	flagBaseURL = flag.String("base-url", "", "URL, including the scheme and final slash, where the root of the website will be\n"+
		"located. This will be used to emit the canonical URL of each page and the sitemap.")
	flagNoSitemap = flag.Bool("no-sitemap", false, "Disable sitemap.txt generation")

	version  = ""
)

func rootRelPath(dir string) string {
	count := strings.Count(dir, "/")
	if count == 0 {
		return "./"
	}
	return strings.Repeat("../", count)
}

func createTagFile(name string) (page string, file *os.File, err error) {
	page = path.Join("tags", name+".html")
	outPath := filepath.Join(outDir, page)
	if err = os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return
	}
	file, err = os.Create(outPath)
	return
}

func renderTag(tag *Tag, params *TmplParams, tagTmpl *template.Template, renamedTmpl *template.Template, pages chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	page, file, err := createTagFile(tag.Name)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	pages <- page
	tagParams := TagTmplParams{
		TmplParams: *params,
		Tag:        tag,
	}
	tagParams.Root = rootRelPath(page)
	if err := tagTmpl.Execute(file, &tagParams); err != nil {
		panic(err)
	}
	for _, name := range tag.RenamedFrom {
		page, file, err := createTagFile(name)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		pages <- page
		tagParams.Root = rootRelPath(page)
		tagParams.PrevName = name
		if err := renamedTmpl.Execute(file, &tagParams); err != nil {
			panic(err)
		}
	}
}

func writeFile(name string, content io.Reader) error {
	path := filepath.Join(outDir, name)
	dir, _ := filepath.Split(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, content); err != nil {
		return err
	}
	return nil
}

func writeAssets() error {
	for _, f := range []File{
		{"main.css", bytes.NewReader(mainCSS)},
		{"openlogo-50.svg", bytes.NewReader(logoSVG)},
		{"favicon.ico", bytes.NewReader(faviconICO)},
	} {
		if err := writeFile(f.name, f.content); err != nil {
			return err
		}
	}
	return nil
}

func writeSitemap(baseURL string, pages <-chan string, wg *sync.WaitGroup) error {
	defer wg.Done()
	file, err := os.Create(filepath.Join(outDir, "sitemap.txt"))
	if err != nil {
		return err
	}
	defer file.Close()
	for page := range pages {
		if _, err := file.WriteString(baseURL + page + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func discardSitemap(pages <-chan string, wg *sync.WaitGroup) error {
	defer wg.Done()
	for range pages {
	}
	return nil
}

func writeManual(tmpl *template.Template, params *TmplParams, path string, pages chan<- string) error {
	file, err := os.Open(manualPath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := ioutil.NewBodyFilterReader(file)
	body := bytes.Buffer{}
	if _, err := io.Copy(&body, reader); err != nil {
		return err
	}
	pages <- path
	manualParams := ManualTmplParams{*params, template.HTML(body.String())}
	manualParams.Root = rootRelPath(path)
	out := bytes.Buffer{}
	if err := tmpl.Execute(&out, &manualParams); err != nil {
		return err
	}
	return writeFile(path, &out)
}

func writeSimplePage(tmpl *template.Template, params TmplParams, path string, root string, pages chan<- string) error {
	file, err := os.Create(filepath.Join(outDir, path))
	if err != nil {
		return err
	}
	defer file.Close()
	if pages != nil {
		pages <- path
	}
	params.Root = root
	return tmpl.Execute(file, &params)
}

func checkErr(err error, msg ...any) {
	if err != nil {
		log.Fatalln(append(append([]any{"ERROR:"}, msg...), err)...)
	}
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	pagesChan := make(chan string, 32)
	sitemapWG := sync.WaitGroup{}
	sitemapWG.Add(1)

	if *flagBaseURL == "" || *flagNoSitemap {
		go discardSitemap(pagesChan, &sitemapWG)
	} else {
		go writeSitemap(*flagBaseURL, pagesChan, &sitemapWG)
	}

	indexTmpl := template.Must(template.New("index").Parse(indexTmplStr))
	tagTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(tagTmplStr))
	renamedTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(renamedTmplStr))
	manualTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(manualTmplStr))
	aboutTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(aboutTmplStr))
	e404Tmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(e404TmplStr))

	listTagsOut := strings.Builder{}
	listTagsCmd := exec.Command("lintian-explain-tags", "--list-tags")
	listTagsCmd.Stderr = os.Stderr
	listTagsCmd.Stdout = &listTagsOut
	checkErr(listTagsCmd.Run(), "list tags:")
	listTagsStr := strings.TrimSpace(listTagsOut.String())
	listTagsLines := strings.Split(listTagsStr, "\n")

	jsonTagsArgs := append([]string{"--format=json"}, listTagsLines...)
	jsonTagsCmd := exec.Command("lintian-explain-tags", jsonTagsArgs...)
	jsonTagsCmd.Stderr = os.Stderr
	jsonTagsOut, err := jsonTagsCmd.StdoutPipe()
	checkErr(err)
	jsonTagsDecoder := json.NewDecoder(jsonTagsOut)
	checkErr(jsonTagsCmd.Start())

	date := time.Now().UTC()
	params := TmplParams{
		BaseURL:     *flagBaseURL,
		DateYear:    date.Year(),
		DateHuman:   date.Format(time.RFC1123),
		DateMachine: date.Format(time.RFC3339),
		Version:     version,
		TagList:     listTagsLines,
	}

	// discard open bracket
	_, err = jsonTagsDecoder.Token()
	checkErr(err)

	tagsWG := sync.WaitGroup{}
	// while the array contains values
	for jsonTagsDecoder.More() {
		var tag Tag
		checkErr(jsonTagsDecoder.Decode(&tag))
		if params.VersionLintian == "" {
			params.VersionLintian = tag.LintianVersion
		}
		tagsWG.Add(1)
		go renderTag(&tag, &params, tagTmpl, renamedTmpl, pagesChan, &tagsWG)
	}

	// discard closing bracket
	_, err = jsonTagsDecoder.Token()
	checkErr(err)

	listTagsJSON, err := json.Marshal(listTagsLines)
	checkErr(err, "marshal listTagsLines:")
	checkErr(writeFile("taglist.json", bytes.NewReader(listTagsJSON)), "write taglist:")
	checkErr(writeAssets(), "write assets:")
	checkErr(writeManual(manualTmpl, &params, "manual/index.html", pagesChan), "write manual:")
	checkErr(writeSimplePage(aboutTmpl, params, "about.html", "./", pagesChan), "write about.html:")
	checkErr(writeSimplePage(indexTmpl, params, "index.html", "./", pagesChan), "write index.html:")
	checkErr(writeSimplePage(e404Tmpl, params, "404.html", "/", nil), "write 404.html:")

	tagsWG.Wait()
	close(pagesChan)
	sitemapWG.Wait()
	if err := jsonTagsCmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			log.Println("WARNING: lintian-explain-tags returned non zero exit status:", exitError.ExitCode())
		} else {
			log.Println("ERROR: running lintian-explain-tags:", err)
		}
	}
}
