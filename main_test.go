package main_test

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	main "github.com/n-peugnet/lintian-ssg"
)

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
// front of the PATH, and finally sets the "--output-dir" CLI flag.
func setup(t *testing.T, lintianExplainTagsOutputs ...any) string {
	checkErr := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	tmpDir := t.TempDir()
	tmpBinDir := filepath.Join(tmpDir, "out")
	checkErr(os.Mkdir(tmpBinDir, 0755))
	file, err := os.Create(filepath.Join(tmpBinDir, "lintian-explain-tags"))
	checkErr(err)
	defer file.Close()
	checkErr(file.Chmod(0755))
	_, err = fmt.Fprintf(file, lintianExplainTagsFmt, lintianExplainTagsOutputs...)
	checkErr(err)
	t.Setenv("PATH", tmpBinDir+":"+os.Getenv("PATH"))

	// Reset command line flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	// Reset CLI args, and add output dir
	outDir := filepath.Join(tmpDir, "out")
	os.Args = []string{os.Args[0], "-o", outDir}
	return outDir
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

func TestEmptyPATH(t *testing.T) {
	setup(t)
	t.Setenv("PATH", "")
	expectPanic(t, `ERROR: list tags: exec: "lintian-explain-tags"`, main.Run)
}

func TestEmptyTagList(t *testing.T) {
	dir := setup(t, "", "[]")
	main.Run()
	buf, _ := exec.Command("ls", "-R", dir).Output()
	t.Logf("%s", buf)
}
