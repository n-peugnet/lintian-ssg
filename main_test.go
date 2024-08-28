package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	main "github.com/n-peugnet/lintian-ssg"
	"github.com/n-peugnet/lintian-ssg/lintian"
	"github.com/n-peugnet/lintian-ssg/version"
)

const lintianVersion = "1.118.0"
const lintianManual = `<body>
MANUAL CONTENT
</body>
`
const lintianExplainTagsFmt = `#!/bin/sh
if test "$1" = "--list-tags"
then
	echo %q
else
	echo %q
fi
`

// setup creates a temporary directory that contains a "bin" dir with an
// executable dummy "lintian-explain-tags" command which is then added in
// front of the PATH, and finally sets the "--output-dir" CLI flag and
// return it.
func setup(t *testing.T, lintianExplainTagsOutputs ...any) fs.FS {
	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}

	// Setup bin directory
	tmpDir := t.TempDir()
	tmpBinDir := filepath.Join(tmpDir, "out")
	checkErr(os.Mkdir(tmpBinDir, 0755))
	binFile, err := os.Create(filepath.Join(tmpBinDir, "lintian-explain-tags"))
	checkErr(err)
	defer binFile.Close()
	checkErr(binFile.Chmod(0755))
	_, err = fmt.Fprintf(binFile, lintianExplainTagsFmt, lintianExplainTagsOutputs...)
	checkErr(err)
	t.Setenv("PATH", tmpBinDir+":"+os.Getenv("PATH"))

	// Setup manual.html
	manualPath := filepath.Join(tmpDir, "manual.html")
	err = os.WriteFile(manualPath, []byte(lintianManual), 0644)
	checkErr(err)
	t.Setenv("LINTIAN_MANUAL_PATH", manualPath)

	// Reset command line flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// Reset CLI args, and add output dir
	outDir := filepath.Join(tmpDir, "out")
	os.Args = []string{os.Args[0], "-o", outDir}

	// Store stdout and stderr
	prevStdout := os.Stdout
	os.Stdout, err = os.Create(filepath.Join(outDir, ".stdout"))
	checkErr(err)
	prevStderr := os.Stderr
	os.Stderr, err = os.Create(filepath.Join(outDir, ".stderr"))
	checkErr(err)
	t.Cleanup(func() {
		os.Stdout = prevStdout
		os.Stderr = prevStderr
	})
	return os.DirFS(outDir)
}

func buildSetupArgs(taglist []string, tags []lintian.Tag) (out []any) {
	var err error
	out = make([]any, 2)
	out[0] = strings.Join(taglist, "\n") + "\n"
	out[1], err = json.Marshal(tags)
	if err != nil {
		panic(err)
	}
	return
}

func expectPanic(t *testing.T, substr string, fn func()) {
	defer func() {
		if err := recover(); err != nil {
			if !strings.Contains(fmt.Sprint(err), substr) {
				t.Fatalf("panic does not contain %q: %q", substr, err)
			}
		} else {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func assertContains(t *testing.T, outDir fs.FS, path string, contents ...string) {
	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	fileContent, err := fs.ReadFile(outDir, path)
	checkErr(err)
	for _, content := range contents {
		i := bytes.Index(fileContent, []byte(content))
		if i == -1 {
			t.Errorf("expected '%s' to be in %s, actual:\n%s", content, path, fileContent)
		}
	}
}

func getHelp(t *testing.T) string {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	startMark := "```--help\n"
	endMark := "```\n"
	start := bytes.Index(readme, []byte(startMark))
	if start == -1 {
		t.Fatal("start mark not found in README.md:", startMark)
	}
	start += len(startMark)
	end := bytes.Index(readme[start:], []byte(endMark))
	if start == -1 {
		t.Fatal("end mark not found in README.md:", endMark)
	}
	return string(readme[start:start+end])
}

func TestBasic(t *testing.T) {
	outDir := setup(t, buildSetupArgs([]string{"test-tag"}, []lintian.Tag{
		{
			Name:           "test-tag",
			NameSpaced:     false,
			Visibility:     lintian.LevelInfo,
			Explanation:    "This is a test.",
			LintianVersion: lintianVersion,
		},
	})...)
	main.Run()

	assertContains(t, outDir, "index.html", `<a href="./tags/test-tag.html">test-tag</a>`)
	assertContains(t, outDir, "manual/index.html", `MANUAL CONTENT`)
	assertContains(t, outDir, "tags/test-tag.html", `<p>This is a test.</p>`)
	assertContains(t, outDir, "taglist.json", `["test-tag"]`)
}

func TestBaseURL(t *testing.T) {
	outDir := setup(t, buildSetupArgs([]string{"test-tag"}, []lintian.Tag{
		{
			Name:           "test-tag",
			NameSpaced:     false,
			Visibility:     lintian.LevelInfo,
			Explanation:    "This is a test.",
			LintianVersion: lintianVersion,
		},
	})...)
	os.Args = append(os.Args, "--base-url=https://lintian.club1.fr")
	main.Run()

	assertContains(t, outDir, "index.html", `<link rel="canonical" href="https://lintian.club1.fr/index.html`)
	assertContains(t, outDir, "manual/index.html", `<link rel="canonical" href="https://lintian.club1.fr/manual/index.html`)
	assertContains(t, outDir, "tags/test-tag.html", `<link rel="canonical" href="https://lintian.club1.fr/tags/test-tag.html`)
	assertContains(t, outDir, "sitemap.txt",
		"https://lintian.club1.fr/about.html",
		"https://lintian.club1.fr/index.html",
		"https://lintian.club1.fr/manual/index.html",
		"https://lintian.club1.fr/tags/test-tag.html",
	)
}

func TestNoSitemap(t *testing.T) {
	outDir := setup(t, buildSetupArgs([]string{"test-tag"}, []lintian.Tag{
		{
			Name:           "test-tag",
			NameSpaced:     false,
			Visibility:     lintian.LevelInfo,
			Explanation:    "This is a test.",
			LintianVersion: lintianVersion,
		},
	})...)
	os.Args = append(os.Args, "--base-url=https://lintian.club1.fr", "--no-sitemap")
	main.Run()

	_, err := outDir.Open("sitemap.txt")
	if err == nil {
		t.Fatal("err should not be nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("expected err to be ErrNotExist, got:", err)
	}
}

func TestHelp(t *testing.T) {
	outDir := setup(t)
	os.Args = append(os.Args, "--help")
	main.Run()
	assertContains(t, outDir, ".stdout", getHelp(t))
}

func TestVersion(t *testing.T) {
	outDir := setup(t)
	os.Args = append(os.Args, "--version")
	main.Run()
	assertContains(t, outDir, ".stdout", version.Number)
}

func TestStats(t *testing.T) {
	outDir := setup(t, buildSetupArgs([]string{"test-tag"}, []lintian.Tag{
		{
			Name:           "test-tag",
			NameSpaced:     false,
			Visibility:     lintian.LevelInfo,
			Explanation:    "This is a test.",
			LintianVersion: lintianVersion,
		},
	})...)
	os.Args = append(os.Args, "--stats")
	main.Run()
	assertContains(t, outDir, ".stdout",
		"number of tags: 1",
		"number of pages: 4",
		"tags list generation CPU time: ",
		"tags json generation CPU time: ",
		"website generation CPU time: ",
		"total duration: ",
	)
}

func TestEmptyPATH(t *testing.T) {
	setup(t)
	t.Setenv("PATH", "")
	expectPanic(t, `ERROR: list tags: exec: "lintian-explain-tags"`, main.Run)
}

func TestEmptyTagList(t *testing.T) {
	setup(t, "", "[]")
	main.Run()
}
