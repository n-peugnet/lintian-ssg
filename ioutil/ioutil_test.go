// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package ioutil_test

import (
	"bytes"
	"io"
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

func TestEmptyBuf(t *testing.T) {
	reader := &bytes.Buffer{}
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 32)

	n, err := filtered.Read(buf)
	if err != io.EOF {
		t.Fatal("unexpected error: ", err)
	}
	if n != 0 {
		t.Fatal("expected n == 0, got:", n)
	}
	expected := []byte{}
	if !bytes.Equal(expected, buf[:n]) {
		t.Fatalf("expected buf[:n] == %q, got: %q", expected, buf[:n])
	}
}

func TestSingleRead(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 32)

	n, err := filtered.Read(buf)
	if err != io.EOF {
		t.Fatal("unexpected error: ", err)
	}
	if n != 30 {
		t.Fatal("expected n == 30, got:", n)
	}
	expected := []byte("testdata3\ntestdata4\ntestdata5\n")
	if !bytes.Equal(expected, buf[:n]) {
		t.Fatalf("expected buf[:n] == %q, got: %q", expected, buf[:n])
	}

	//following reads
	n, err = filtered.Read(buf)
	if err != io.EOF {
		t.Fatal("unexpected error: ", err)
	}
	if n != 0 {
		t.Fatal("expected n == 0, got:", n)
	}
}

func TestTwoReads(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 16)

	// first read
	n, err := filtered.Read(buf)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	if n != 16 {
		t.Fatal("expected n == 16, got:", n)
	}
	expected := []byte("testdata3\ntestda")
	if !bytes.Equal(expected, buf) {
		t.Fatalf("expected buf == %q, got: %q", expected, buf)
	}

	//second read
	n, err = filtered.Read(buf)
	if err != io.EOF {
		t.Fatal("unexpected error: ", err)
	}
	if n != 14 {
		t.Fatal("expected n == 14, got:", n)
	}
	expected = []byte("ta4\ntestdata5\n")
	if !bytes.Equal(expected, buf[:n]) {
		t.Fatalf("expected buf[:n] == %q, got: %q", expected, buf[:n])
	}
}

func TestFullRead(t *testing.T) {
	reader := bytes.NewReader(data)
	filtered := ioutil.NewBodyFilterReader(reader)

	buf := make([]byte, 30)

	n, err := filtered.Read(buf)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}
	if n != 30 {
		t.Fatal("expected n == 30, got:", n)
	}
	expected := []byte("testdata3\ntestdata4\ntestdata5\n")
	if !bytes.Equal(expected, buf[:n]) {
		t.Fatalf("expected buf[:n] == %q, got: %q", expected, buf[:n])
	}

	//following reads
	n, err = filtered.Read(buf)
	if err != io.EOF {
		t.Fatal("unexpected error: ", err)
	}
	if n != 0 {
		t.Fatal("expected n == 0, got:", n)
	}
}
