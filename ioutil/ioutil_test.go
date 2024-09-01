// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package ioutil_test

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/iotest"

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

func assertRead(t *testing.T, r io.Reader, buf []byte, expected string, expectedErr error, msg string) {
	n, err := r.Read(buf)
	if err != expectedErr {
		t.Fatalf("%s: unexpected err == %v, got: %v", msg, expectedErr, err)
	}
	if n != len(expected) {
		t.Fatalf("%s: expected n == %d, got: %v", msg, len(expected), n)
	}
	if !bytes.Equal([]byte(expected), buf[:n]) {
		t.Fatalf("%s: expected buf[:n] == %q, got: %q", msg, expected, buf[:n])
	}
}

func TestBodyReaderEmptyBuf(t *testing.T) {
	reader := &bytes.Buffer{}
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 32)
	assertRead(t, filtered, buf, "", io.EOF, "first read")
}

func TestBodyReaderSingleRead(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 32)
	assertRead(t, filtered, buf, "testdata3\ntestdata4\ntestdata5\n", io.EOF, "first read")
	assertRead(t, filtered, buf, "", io.EOF, "following read")
}

func TestBodyReaderTwoReads(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 16)
	assertRead(t, filtered, buf, "testdata3\ntestda", nil, "first read")
	assertRead(t, filtered, buf, "ta4\ntestdata5\n", io.EOF, "second read")
}

func TestBodyReaderFullRead(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 30)
	assertRead(t, filtered, buf, "testdata3\ntestdata4\ntestdata5\n", nil, "first read")
	assertRead(t, filtered, buf, "", io.EOF, "following read")
}

func TestBodyReaderLongLines(t *testing.T) {
	data := []byte(`testdata0
testdata1 testdata2
<body>
testdata3 testdata4 testdata5
</body>
testdata6 testdata7 testdata8`)
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 8)
	assertRead(t, filtered, buf, "testdata", nil, "first read")
	assertRead(t, filtered, buf, "3 testda", nil, "second read")
	assertRead(t, filtered, buf, "ta4 test", nil, "third read")
	assertRead(t, filtered, buf, "data5\n", io.EOF, "fourth read")
}

func TestBodyReaderTimeoutReader(t *testing.T) {
	data := &bytes.Buffer{}
	data.Write(make([]byte, 4070))
	data.WriteString(`
<body>
testdata1
testdata2
</body>
`)
	reader := iotest.TimeoutReader(data)
	filtered := ioutil.NewBodyFilterReaderSize(reader, 4096)

	buf := make([]byte, 8)
	assertRead(t, filtered, buf, "testdata", nil, "first read")
	assertRead(t, filtered, buf, "1\n", iotest.ErrTimeout, "second read")
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

func TestWriteFileErrReader(t *testing.T) {
	name := "test/file.txt"
	outDir := t.TempDir()
	err := ioutil.WriteFile(outDir, name, iotest.ErrReader(io.ErrUnexpectedEOF))
	if err != io.ErrUnexpectedEOF {
		t.Fatalf("expected %v, got: %v", io.ErrUnexpectedEOF, err)
	}
}
