package ioutil

import (
	"bufio"
	"bytes"
	"io"
)

// BodyFilterReader is a io.Reader wrapper that returns only the content of the
// <body></body> HTML tag. It is a real HTML parser and is very dumb, as it can
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
