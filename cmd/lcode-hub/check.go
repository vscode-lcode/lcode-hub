package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

func hasRunning(addr string) (err error) {
	defer err2.Handle(&err)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp := try.To1(client.Get(fmt.Sprintf("http://%s/proc/lcode-version", addr)))
	name := string(try.To1(io.ReadAll(resp.Body)))

	if n := f.Name(); n != name {
		err2.Throwf("version is diffrence. running is %s, want %s", name, n)
	}

	logger.Println(name, "already running")
	return
}
