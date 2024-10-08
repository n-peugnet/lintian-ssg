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

package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/n-peugnet/lintian-ssg/ioutil"
	"github.com/n-peugnet/lintian-ssg/lintian"
	"github.com/n-peugnet/lintian-ssg/markdown"
	"github.com/n-peugnet/lintian-ssg/version"
)

const (
	manualPath   = "/usr/share/doc/lintian/lintian.html"
	sourceURLFmt = "https://salsa.debian.org/lintian/lintian/-/blob/%s/tags/%s.tag"
)

type tmplParams struct {
	DateYear       int
	DateHuman      string
	DateMachine    string
	BaseURL        string
	Root           string
	Version        string
	VersionLintian string
	FooterHTML     template.HTML
}

type indexTmplParams struct {
	tmplParams
	TagList []string
}

type manualTmplParams struct {
	tmplParams
	Manual template.HTML
}

type tagTmplParams struct {
	tmplParams
	*lintian.Tag
	PrevName string
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

	start = time.Now()
)

var (
	flagBaseURL   string
	flagFooter    string
	flagHelp      bool
	flagNoSitemap bool
	flagOutDir    string
	flagStats     bool
	flagVersion   bool
)

const (
	flagBaseURLHelp = `URL, including the scheme, where the root of the website will be located.
        This will be used in the sitemap and in the canonical URL of each page.`
	flagFooterHelp    = "Text to add to the footer, inline Markdown elements will be parsed."
	flagHelpHelp      = "Show this help and exit."
	flagNoSitemapHelp = "Disable sitemap.txt generation."
	flagOutDirHelp    = "Path of the directory where to output the generated website."
	flagOutDirDef     = "out"
	flagStatsHelp     = "Display some statistics."
	flagVersionHelp   = "Show version and exit."
)

func usage() {
	var output io.Writer
	if flagHelp {
		output = os.Stdout
	} else {
		output = flag.CommandLine.Output()
	}
	fmt.Fprintf(output, `Usage of lintian-ssg:
  --base-url string
        %s
  --footer string
        %s
  -h, --help
        %s
  --no-sitemap
        %s
  -o, --output-dir string
        %s (default %q)
  --stats
        %s
  --version
        %s
`,
		flagBaseURLHelp,
		flagFooterHelp,
		flagHelpHelp,
		flagNoSitemapHelp,
		flagOutDirHelp, flagOutDirDef,
		flagStatsHelp,
		flagVersionHelp,
	)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func rootRelPath(dir string) string {
	count := strings.Count(dir, "/")
	if count == 0 {
		return "./"
	}
	return strings.Repeat("../", count)
}

func createTagFile(name string) (page string, file *os.File, err error) {
	page = path.Join("tags", name+".html")
	outPath := filepath.Join(flagOutDir, page)
	if err = os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return
	}
	file, err = os.Create(outPath)
	return
}

func renderTag(tag *lintian.Tag, params *tmplParams, tagTmpl *template.Template, renamedTmpl *template.Template, pages chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	page, file, err := createTagFile(tag.Name)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	pages <- page
	tagParams := tagTmplParams{
		tmplParams: *params,
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

func writeAssets() error {
	files := []struct {
		name    string
		content io.Reader
	}{
		{"main.css", bytes.NewReader(mainCSS)},
		{"openlogo-50.svg", bytes.NewReader(logoSVG)},
		{"favicon.ico", bytes.NewReader(faviconICO)},
	}
	for _, f := range files {
		if err := ioutil.WriteFile(flagOutDir, f.name, f.content); err != nil {
			return err
		}
	}
	return nil
}

func writeSitemap(baseURL string, pages []string) error {
	file, err := os.Create(filepath.Join(flagOutDir, "sitemap.txt"))
	if err != nil {
		return err
	}
	defer file.Close()
	sort.Strings(pages)
	builder := strings.Builder{}
	builder.Grow(len(pages) * 32)
	for _, page := range pages {
		builder.WriteString(baseURL + page + "\n")
	}
	if _, err := file.WriteString(builder.String()); err != nil {
		return err
	}
	return nil
}

func writeManual(tmpl *template.Template, params *tmplParams, path string, pages chan<- string) error {
	file, err := os.Open(getEnv("LINTIAN_MANUAL_PATH", manualPath))
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
	manualParams := manualTmplParams{*params, template.HTML(body.String())}
	manualParams.Root = rootRelPath(path)
	out := bytes.Buffer{}
	if err := tmpl.Execute(&out, &manualParams); err != nil {
		return err
	}
	return ioutil.WriteFile(flagOutDir, path, &out)
}

func writeSimplePage(tmpl *template.Template, params any, path string, pages chan<- string) error {
	file, err := os.Create(filepath.Join(flagOutDir, path))
	if err != nil {
		return err
	}
	defer file.Close()
	if pages != nil {
		pages <- path
	}
	return tmpl.Execute(file, params)
}

func handlePages(pages <-chan string, count *int, wg *sync.WaitGroup) {
	defer wg.Done()
	s := make([]string, 0, 2048)
	for page := range pages {
		s = append(s, page)
	}
	if flagBaseURL != "" && !flagNoSitemap {
		if err := writeSitemap(flagBaseURL, s); err != nil {
			panic(err)
		}
	}
	*count = len(s)
}

func withRoot(params tmplParams, root string) tmplParams {
	params.Root = root
	return params
}

func checkErr(err error, msg ...any) {
	if err != nil {
		panic(fmt.Sprintln(append(append([]any{"ERROR:"}, msg...), err)...))
	}
}

func Run() {
	log.SetFlags(0)
	flag.StringVar(&flagBaseURL, "base-url", "", flagBaseURLHelp)
	flag.StringVar(&flagFooter, "footer", "", flagFooterHelp)
	flag.BoolVar(&flagHelp, "h", false, flagHelpHelp)
	flag.BoolVar(&flagHelp, "help", false, flagHelpHelp)
	flag.BoolVar(&flagNoSitemap, "no-sitemap", false, flagNoSitemapHelp)
	flag.StringVar(&flagOutDir, "o", flagOutDirDef, flagOutDirHelp)
	flag.StringVar(&flagOutDir, "output-dir", flagOutDirDef, flagOutDirHelp)
	flag.BoolVar(&flagStats, "stats", false, flagStatsHelp)
	flag.BoolVar(&flagVersion, "version", false, flagVersionHelp)
	flag.Usage = usage
	flag.Parse()

	if flagHelp {
		flag.Usage()
		return
	}
	if flagVersion {
		fmt.Println(version.Number)
		return
	}
	if flagBaseURL != "" && !strings.HasSuffix(flagBaseURL, "/") {
		flagBaseURL += "/"
	}

	checkErr(os.MkdirAll(flagOutDir, 0755), "create out dir:")

	pagesChan := make(chan string, 32)
	pagesWG := sync.WaitGroup{}
	pagesWG.Add(1)
	var pagesCount int
	go handlePages(pagesChan, &pagesCount, &pagesWG)

	indexTmpl := template.Must(template.New("index").Parse(indexTmplStr))
	tagTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(tagTmplStr))
	renamedTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(renamedTmplStr))
	manualTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(manualTmplStr))
	aboutTmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(aboutTmplStr))
	e404Tmpl := template.Must(template.Must(indexTmpl.Clone()).Parse(e404TmplStr))

	jsonTagsCmd := exec.Command("lintian-explain-tags", "--format=json")
	jsonTagsCmd.Stderr = os.Stderr
	jsonTagsOut, err := jsonTagsCmd.StdoutPipe()
	checkErr(err)
	jsonTagsDecoder := json.NewDecoder(jsonTagsOut)
	checkErr(jsonTagsCmd.Start(), "lintian-explain-tags --format=json:")

	date := time.Now().UTC()
	params := tmplParams{
		BaseURL:     flagBaseURL,
		DateYear:    date.Year(),
		DateHuman:   date.Format(time.RFC1123),
		DateMachine: date.Format(time.RFC3339),
		Version:     version.Number,
		FooterHTML:  markdown.ToHTML(flagFooter, markdown.StyleInline),
	}

	tagList := make([]string, 0, 2048)

	// discard open bracket
	_, err = jsonTagsDecoder.Token()
	checkErr(err)

	tagsWG := sync.WaitGroup{}
	// while the array contains values
	for jsonTagsDecoder.More() {
		var tag lintian.Tag
		checkErr(jsonTagsDecoder.Decode(&tag))
		if params.VersionLintian == "" {
			params.VersionLintian = tag.LintianVersion
		}
		tagsWG.Add(1)
		go renderTag(&tag, &params, tagTmpl, renamedTmpl, pagesChan, &tagsWG)
		tagList = append(tagList, tag.Name)
	}

	// discard closing bracket
	_, err = jsonTagsDecoder.Token()
	checkErr(err)

	tagListJSON, err := json.Marshal(tagList)
	checkErr(err, "marshal tagList:")
	checkErr(ioutil.WriteFile(flagOutDir, "taglist.json", bytes.NewReader(tagListJSON)), "write taglist:")
	checkErr(writeAssets(), "write assets:")
	checkErr(writeManual(manualTmpl, &params, "manual/index.html", pagesChan), "write manual:")
	indexParams := indexTmplParams{withRoot(params, "./"), tagList}
	checkErr(writeSimplePage(indexTmpl, indexParams, "index.html", pagesChan), "write index.html:")
	checkErr(writeSimplePage(aboutTmpl, withRoot(params, "./"), "about.html", pagesChan), "write about.html:")
	checkErr(writeSimplePage(e404Tmpl, withRoot(params, "/"), "404.html", nil), "write 404.html:")

	tagsWG.Wait()
	close(pagesChan)
	if err := jsonTagsCmd.Wait(); err != nil {
		log.Println("WARNING: lintian-explain-tags --format=json:", err)
	}

	pagesWG.Wait()
	if flagStats {
		usage := syscall.Rusage{}
		checkErr(syscall.Getrusage(syscall.RUSAGE_SELF, &usage), "get resources usage:")
		fmt.Printf(`number of tags: %d
number of pages: %d
tags json generation CPU time: %v (user: %v sys: %v)
website generation CPU time: %v (user: %v sys: %v)
total duration: %v
`,
			len(tagList),
			pagesCount,
			(jsonTagsCmd.ProcessState.UserTime() + jsonTagsCmd.ProcessState.SystemTime()).Round(time.Millisecond),
			jsonTagsCmd.ProcessState.UserTime().Round(time.Millisecond),
			jsonTagsCmd.ProcessState.SystemTime().Round(time.Millisecond),
			time.Duration(usage.Utime.Nano()+usage.Stime.Nano()).Round(time.Millisecond),
			time.Duration(usage.Utime.Nano()).Round(time.Millisecond),
			time.Duration(usage.Stime.Nano()).Round(time.Millisecond),
			time.Now().Sub(start).Round(time.Millisecond),
		)
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	Run()
}
