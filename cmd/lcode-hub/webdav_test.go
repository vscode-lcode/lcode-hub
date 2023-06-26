package main

import (
	"regexp"
	"testing"

	"github.com/lainio/err2/assert"
)

func TestHostMatcher(t *testing.T) {
	m := regexp.MustCompile(`^(.+)\.lo`)
	s := m.FindStringSubmatch("webdav.lo.shynome.com")
	assert.Equal(len(s), 2)
	assert.Equal(s[1], "webdav")
}
