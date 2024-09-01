// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package ioutil_test

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/n-peugnet/lintian-ssg/ioutil"
)

var data = []byte(`testdata0
testdata1
testdata2
<body>
testdata3
testdata4
testdata5
</body>
testdata6
testdata7
testdata8`)

func assertRead(t *testing.T, r io.Reader, buf []byte, expected string, expectedErr error) {
	n, err := r.Read(buf)
	if err != expectedErr {
		t.Fatalf("unexpected err == %v, got: %v", expectedErr, err)
	}
	if n != len(expected) {
		t.Fatalf("expected n == %d, got: %v", len(expected), n)
	}
	if !bytes.Equal([]byte(expected), buf[:n]) {
		t.Fatalf("expected buf[:n] == %q, got: %q", expected, buf[:n])
	}
}

func TestBodyReaderEmptyBuf(t *testing.T) {
	reader := &bytes.Buffer{}
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 32)
	assertRead(t, filtered, buf, "", io.EOF)
}

func TestBodyReaderSingleRead(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 32)

	// first read
	assertRead(t, filtered, buf, "testdata3\ntestdata4\ntestdata5\n", io.EOF)

	// following reads
	assertRead(t, filtered, buf, "", io.EOF)
}

func TestBodyReaderTwoReads(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 16)

	// first read
	assertRead(t, filtered, buf, "testdata3\ntestda", nil)

	//second read
	assertRead(t, filtered, buf, "ta4\ntestdata5\n", io.EOF)
}

func TestBodyReaderFullRead(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 30)

	assertRead(t, filtered, buf, "testdata3\ntestdata4\ntestdata5\n", nil)

	// following reads
	assertRead(t, filtered, buf, "", io.EOF)
}


func TestWriteFileBasic(t *testing.T) {
	name := "test/file.txt"
	expected := []byte("Hello world!\n")
	outDir := t.TempDir()
	content := bytes.NewBuffer(expected)
	if err := ioutil.WriteFile(outDir, name, content); err != nil {
		t.Fatal("unexpected error:", err)
	}
	dirFS := os.DirFS(outDir)
	actual, err := fs.ReadFile(dirFS, name)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if !bytes.Equal(expected, actual) {
		t.Fatalf("expected %q, got: %q", expected, actual)
	}
}

func TestWriteFilePermissionDenied(t *testing.T) {
	name := "test/file.txt"
	outDir := t.TempDir()
	if err := os.Chmod(outDir, 0400); err != nil {
		t.Fatal(err)
	}
	content := bytes.NewBuffer([]byte("Hello world!\n"))
	if err := ioutil.WriteFile(outDir, name, content); err == nil {
		t.Fatal("expected error, got:", err)
	}
}

func TestWriteFileTooLongName(t *testing.T) {
	name := "test/veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryloooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooonnnnnnnnnnnngname.txt"
	outDir := t.TempDir()
	content := bytes.NewBuffer([]byte("Hello world!\n"))
	if err := ioutil.WriteFile(outDir, name, content); err == nil {
		t.Fatal("expected error, got:", err)
	}
}
