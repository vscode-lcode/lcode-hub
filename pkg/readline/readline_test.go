package readline

import (
	"bytes"
	"io"
	"testing"

	"github.com/lainio/err2/assert"
	"github.com/lainio/err2/try"
)

func TestReadLine(t *testing.T) {
	var r = bytes.NewBufferString("\n>2: eeeee\n>2: 777\n99999")
	rr := New(r)
	s := try.To1(rr.ReadLine())
	assert.Equal(string(s), "eeeee")
	t.Log(s)
	s = try.To1(rr.ReadLine())
	assert.Equal(string(s), "777")
	t.Log(s)
	s, err := rr.ReadLine()
	assert.Equal(err, io.EOF)
}
