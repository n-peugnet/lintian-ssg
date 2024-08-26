package main_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	main "github.com/n-peugnet/lintian-ssg"
	"github.com/n-peugnet/lintian-ssg/lintian"
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
	manualFile, err := os.Create(manualPath)
	checkErr(err)
	defer manualFile.Close()
	_, err = manualFile.WriteString(lintianManual)
	checkErr(err)
	t.Setenv("LINTIAN_MANUAL_PATH", manualPath)

	// Reset command line flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// Reset CLI args, and add output dir
	outDir := filepath.Join(tmpDir, "out")
	os.Args = []string{os.Args[0], "-o", outDir}
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

func assertContains(t *testing.T, outDir fs.FS, path string, contents []string) {
	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	file, err := outDir.Open(path)
	checkErr(err)
	fileContent, err := io.ReadAll(file)
	checkErr(err)
	for _, content := range contents {
		i := bytes.Index(fileContent, []byte(content))
		if i == -1 {
			t.Errorf("expected '%s' to be in %s, actual: %s", content, path, fileContent)
		}
	}
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

	assertContains(t, outDir, "index.html", []string{`<a href="./tags/test-tag.html">test-tag</a>`})
	assertContains(t, outDir, "manual/index.html", []string{`MANUAL CONTENT`})
	assertContains(t, outDir, "tags/test-tag.html", []string{`<p>This is a test.</p>`})
	assertContains(t, outDir, "taglist.json", []string{`["test-tag"]`})
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
