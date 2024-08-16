// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package ioutil

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
)

// BodyFilterReader is a io.Reader wrapper that returns only the content of the
// <body></body> HTML tag. It is NOT a real HTML parser and is very dumb, as it can
// only work if the start and end tags are each on a single line and have no
// attributes.
type BodyFilterReader struct {
	r        *bufio.Reader
	rest     []byte
	openTag  bool
	closeTag bool
}

// NewBodyFilterReader returns a new BodyFilterReader
func NewBodyFilterReader(reader io.Reader) *BodyFilterReader {
	return &BodyFilterReader{r: bufio.NewReader(reader)}
}

func (r *BodyFilterReader) Read(buf []byte) (int, error) {
	if r.closeTag {
		return 0, io.EOF
	}
	for !r.openTag {
		buf, err := r.r.ReadBytes('\n')
		if err != nil {
			return 0, err
		}
		if bytes.Equal(buf, []byte("<body>\n")) {
			r.openTag = true
			break
		}
	}
	count := 0
	for count < len(buf) {
		if r.rest != nil {
			n := copy(buf[count:], r.rest)
			count += n
			if n != len(r.rest) {
				r.rest = r.rest[n:]
				continue
			}
			r.rest = nil
		}
		tmp, err := r.r.ReadBytes('\n')
		if err != nil {
			return count, err;
		}
		if bytes.Equal(tmp, []byte("</body>\n")) {
			r.closeTag = true
			return count, io.EOF
		}
		n := copy(buf[count:], tmp)
		count += n
		if n != len(tmp) {
			r.rest = tmp[n:]
			continue
		}
	}
	return count, nil
}

// WriteFile creates or override a file in outDir while creating required directories.
func WriteFile(outDir string, name string, content io.Reader) error {
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
