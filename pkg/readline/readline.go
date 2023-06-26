package readline

import (
	"io"
	"unicode"
)

type ReadLine struct {
	io.Reader
	cursor *Cursor
}

func New(r io.Reader) *ReadLine {
	var c Cursor
	return &ReadLine{
		Reader: r,
		cursor: &c,
	}
}

const prefix = "\n>2: "
const prefixSize = len(prefix)
const br = byte('\n')

type Cursor [prefixSize]byte

func (c *Cursor) Push(b byte) {
	v := append(c[:], b)
	copy(c[:], v[1:])
}

func (r *ReadLine) ReadLine() (line []byte, err error) {
	var one [1]byte
	cursor := r.cursor
	for {
		n, err := r.Read(one[:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			continue
		}
		if one[0] != br && !unicode.IsGraphic(rune(one[0])) {
			continue
		}
		cursor.Push(one[0])
		if string(cursor[:]) == prefix {
			for {
				n, err := r.Read(one[:])
				if err != nil {
					return nil, err
				}
				if n == 0 {
					continue
				}
				if one[0] == br {
					cursor.Push(br)
					return line, nil
				}
				line = append(line, one[0])
			}
		}
	}
}
