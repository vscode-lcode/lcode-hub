package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/lcode-hub/pkg/readline"
)

type Stderr struct {
	net.Conn
	Host    string
	Workdir string
	closed  chan any
}

type Supervisor struct {
	HostMatcher *regexp.Regexp

	handlers map[uint32]*Handler
	nextID   uint32
	locker   *sync.RWMutex
}

type Handler struct {
	Header
	ch chan *Stderr
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		HostMatcher: regexp.MustCompile(defaultHostMatcher),

		handlers: make(map[uint32]*Handler),
		locker:   &sync.RWMutex{},
	}
}

func (dav *Supervisor) NewHandler() *Handler {
	id := dav.genID()

	dav.locker.Lock()
	defer dav.locker.Unlock()
	var hdr Header
	hdr.encode(id, 0)
	h := &Handler{
		Header: hdr,
		ch:     make(chan *Stderr),
	}
	dav.handlers[id] = h
	return h
}

func (sup *Supervisor) genID() uint32 {
	sup.locker.RLock()
	defer sup.locker.RUnlock()
	for {
		id := sup.nextID
		sup.nextID++
		if client, ok := sup.handlers[id]; !ok || client == nil {
			return id
		}
	}
}

func (dav *Supervisor) StderrPipe(conn net.Conn) {
	var done = make(chan any)
	defer err2.Catch(func(err error) {
		conn.Close()
		close(done)
	})
	r2 := io.MultiReader(bytes.NewReader([]byte("\n")), conn)
	r := readline.New(r2)
	var init = &sync.Once{}
	line := try.To1(r.ReadLine())
	var h Header
	{
		line := try.To1(hex.DecodeString(string(line)))
		if len(line) < 8 {
			err2.Throwf("got header failed")
		}
		h = Header(line[:8])
	}
	for {
		line := try.To1(r.ReadLine())
		go func(rawLine []byte) {
			line := getShowedString(string(rawLine))
			fmt.Fprintln(conn, line)
			init.Do(func() {
				defer err2.Catch(func(err error) {
					conn.Close()
				})
				link := line[4:]
				u := try.To1(url.Parse(link))
				host := try.To1(getHost(dav.HostMatcher, u.Host))
				stderr := &Stderr{
					Conn:    conn,
					Host:    host,
					Workdir: u.Path,
					closed:  done,
				}
				try.To(dav.dialHandler(h, stderr, conn))
			})
		}(line)
	}
}

func (dav *Supervisor) dialHandler(h Header, stderr *Stderr, conn net.Conn) (err error) {
	defer err2.Handle(&err)

	dav.locker.RLock()
	handler, ok := dav.handlers[h.ID()]
	dav.locker.RUnlock()

	if !ok || handler == nil {
		err2.Throwf("handler not found")
	}
	if h.MagicCode() != handler.MagicCode() {
		err2.Throwf("handler is reused")
	}

	handler.ch <- stderr

	// 清理已处理的handler
	dav.locker.Lock()
	defer dav.locker.Unlock()
	dav.handlers[h.ID()] = nil

	return
}

func getShowedString(l string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}, l)
}

type Header [8]byte

func (h *Header) ID() uint32 {
	return binary.BigEndian.Uint32(h[0:4])
}

func (h *Header) MagicCode() uint32 {
	return binary.BigEndian.Uint32(h[4:8])
}

func (h *Header) encode(id uint32, code uint32) {
	if code == 0 {
		code = rand.Uint32()
	}
	binary.BigEndian.PutUint32(h[0:4], id)
	binary.BigEndian.PutUint32(h[4:8], code)
}

func (h *Header) String() string {
	return hex.EncodeToString(h[:])
}
