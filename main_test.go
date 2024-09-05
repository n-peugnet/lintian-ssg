// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
if test "$1" = "--format=json"
then
	# %[1]d workaround for https://github.com/golang/go/issues/45742
	echo %[2]q
	exit %[1]d
fi
`

// e is a shorthand for [regexp.QuoteMeta] which returns a string that escapes
// all regular expression metacharacters inside the argument text;
// the returned string is a regular expression matching the literal text.
var e = regexp.QuoteMeta

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
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.PanicOnError)
	flag.CommandLine.Usage = func() { flag.Usage() }

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
	log.SetOutput(os.Stderr)
	t.Cleanup(func() {
		os.Stdout = prevStdout
		os.Stderr = prevStderr
	})
	return os.DirFS(outDir)
}

func buildSetupArgs(exitCode int, tags []lintian.Tag) []any {
	var err error
	out := make([]any, 2)
	out[0] = exitCode
	out[1], err = json.Marshal(tags)
	if err != nil {
		panic(err)
	}
	return out
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

// assertContains verifies for each of the given needles that they are in
// the content of the file (haystack) located at the given path in outDir.
func assertContains(t *testing.T, outDir fs.FS, path string, needles ...string) {
	content, err := fs.ReadFile(outDir, path)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range needles {
		i := bytes.Index(content, []byte(needle))
		if i == -1 {
			t.Errorf("expected '%s' to be in %s, actual:\n%s", needle, path, content)
		}
	}
}

// assertRegexp verifies for each of the given expressions that they are
// matching the content of the file located at the given path in outDir.
func assertRegexp(t *testing.T, outDir fs.FS, path string, expressions ...string) {
	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	content, err := fs.ReadFile(outDir, path)
	checkErr(err)
	for _, expression := range expressions {
		matched, err := regexp.Match(expression, content)
		checkErr(err)
		if !matched {
			t.Errorf("expected %s to match '%s', actual:\n%s", path, expression, content)
		}
	}
}

// assertSame verifies that the file located athe given path in outDir is
// the same as the one at the expected location.
func assertSame(t *testing.T, outDir fs.FS, path string, expected string) {
	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	content, err := fs.ReadFile(outDir, path)
	checkErr(err)
	expectedContent, err := os.ReadFile(filepath.Clean(expected))
	checkErr(err)
	if !bytes.Equal(content, expectedContent) {
		t.Errorf("expected %s to be the same as %s", path, expected)
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
		t.Fatalf("start mark not found in README.md: %q", startMark)
	}
	start += len(startMark)
	end := bytes.Index(readme[start:], []byte(endMark))
	if end == -1 {
		t.Fatalf("end mark not found in README.md: %q", endMark)
	}
	end += start
	return string(readme[start:end])
}

func TestBasic(t *testing.T) {
	outDir := setup(t, buildSetupArgs(0, []lintian.Tag{
		{
			Name:           "test-tag",
			NameSpaced:     false,
			Visibility:     lintian.LevelInfo,
			Explanation:    "This is a test.",
			LintianVersion: lintianVersion,
			RenamedFrom:    []string{"previous-tag"},
		},
		{
			Name:           "nested/test/tag",
			NameSpaced:     true,
			Visibility:     lintian.LevelError,
			Explanation:    "This is a nested test.",
			LintianVersion: lintianVersion,
		},
	})...)
	main.Run()

	assertContains(t, outDir, "index.html",
		`<a href="./tags/test-tag.html">test-tag</a>`,
		`<a href="./tags/nested/test/tag.html">nested/test/tag</a>`,
		`<link rel="stylesheet" href="./main.css">`,
	)
	assertContains(t, outDir, "manual/index.html",
		`MANUAL CONTENT`,
		`<link rel="stylesheet" href="../main.css">`,
	)
	assertContains(t, outDir, "tags/test-tag.html",
		`<p>This is a test.</p>`,
		`<link rel="stylesheet" href="../main.css">`,
	)
	assertContains(t, outDir, "tags/previous-tag.html",
		`<a href="../tags/test-tag.html"><code>test-tag</code></a>`,
		`<link rel="stylesheet" href="../main.css">`,
	)
	assertContains(t, outDir, "tags/nested/test/tag.html",
		`<p>This is a nested test.</p>`,
		`<link rel="stylesheet" href="../../../main.css">`,
	)
	assertContains(t, outDir, "taglist.json", `["test-tag","nested/test/tag"]`)
	assertSame(t, outDir, "main.css", "assets/main.css")
	assertSame(t, outDir, "favicon.ico", "assets/favicon.ico")
	assertSame(t, outDir, "openlogo-50.svg", "assets/openlogo-50.svg")
}

func TestJSONTagsError(t *testing.T) {
	outDir := setup(t, buildSetupArgs(1, []lintian.Tag{})...)
	main.Run()
	assertContains(t, outDir, ".stderr", "WARNING: lintian-explain-tags --format=json: exit status 1")
}

func TestBaseURL(t *testing.T) {
	outDir := setup(t, buildSetupArgs(0, []lintian.Tag{
		{
			Name:           "test-tag",
			NameSpaced:     false,
			Visibility:     lintian.LevelInfo,
			Explanation:    "This is a test.",
			LintianVersion: lintianVersion,
			RenamedFrom:    []string{"previous-tag"},
		},
	})...)
	os.Args = append(os.Args, "--base-url=https://lintian.club1.fr")
	main.Run()

	assertContains(t, outDir, "index.html", `<link rel="canonical" href="https://lintian.club1.fr/index.html`)
	assertContains(t, outDir, "manual/index.html", `<link rel="canonical" href="https://lintian.club1.fr/manual/index.html`)
	assertContains(t, outDir, "tags/test-tag.html", `<link rel="canonical" href="https://lintian.club1.fr/tags/test-tag.html`)
	assertContains(t, outDir, "tags/previous-tag.html", `<link rel="canonical" href="https://lintian.club1.fr/tags/previous-tag.html`)
	assertContains(t, outDir, "sitemap.txt",
		"https://lintian.club1.fr/about.html",
		"https://lintian.club1.fr/index.html",
		"https://lintian.club1.fr/manual/index.html",
		"https://lintian.club1.fr/tags/test-tag.html",
		"https://lintian.club1.fr/tags/previous-tag.html",
	)
}

func TestNoSitemap(t *testing.T) {
	outDir := setup(t, buildSetupArgs(0, []lintian.Tag{
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

func TestNonExistingFlag(t *testing.T) {
	outDir := setup(t)
	os.Args = append(os.Args, "--non-existing-flag")
	expectPanic(t, "-non-existing-flag", main.Run)
	assertContains(t, outDir, ".stderr", getHelp(t))
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
	outDir := setup(t, buildSetupArgs(0, []lintian.Tag{
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
	assertRegexp(t, outDir, ".stdout",
		e("number of tags: 1"),
		e("number of pages: 4"),
		`tags json generation CPU time: (\d.)?\d+m?s \(user: (\d.)?\d+m?s sys: (\d.)?\d+m?s\)`,
		`website generation CPU time: (\d.)?\d+m?s \(user: (\d.)?\d+m?s sys: (\d.)?\d+m?s\)`,
		`total duration: (\d.)?\d+m?s`,
	)
}

func TestEmptyPATH(t *testing.T) {
	setup(t)
	t.Setenv("PATH", "")
	expectPanic(t, `ERROR: lintian-explain-tags --format=json: exec: "lintian-explain-tags"`, main.Run)
}

func TestEmptyTagList(t *testing.T) {
	setup(t, 0, "[]")
	main.Run()
}
