package main

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/bash"
	bwebdav "github.com/vscode-lcode/bash/webdav"
	"golang.org/x/net/webdav"
)

type Session struct {
	http.Handler
	Host    string
	Workdir string
	Stderr  chan net.Conn
	Session bash.Session
}

func (s *Session) String() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Workdir)
}

type WebdavHub struct {
	HostMatcher *regexp.Regexp
	connections map[string]map[string]*Session
	locker      *sync.RWMutex
}

var _ http.Handler = (*WebdavHub)(nil)

func NewWebdavHub() *WebdavHub {
	return &WebdavHub{
		HostMatcher: regexp.MustCompile(defaultHostMatcher),
		locker:      &sync.RWMutex{},
		connections: make(map[string]map[string]*Session),
	}
}

func (dav *WebdavHub) NewSession(host, path string, sess bash.Session) (*Session, error) {
	dav.locker.Lock()
	defer dav.locker.Unlock()
	handler := &webdav.Handler{
		FileSystem: bwebdav.NewFileSystem(sess),
		LockSystem: webdav.NewMemLS(),
	}
	s := &Session{
		Handler: handler,
		Host:    host, Workdir: path,
		Session: sess,
	}
	m, ok := dav.connections[host]
	if !ok {
		m = make(map[string]*Session)
		dav.connections[host] = m
	}
	if mm, ok := m[path]; ok && mm != nil {
		return nil, fmt.Errorf("%s %s session is already opened", host, path)
	}
	m[path] = s
	return s, nil
}

func (dav *WebdavHub) RemoveSession(s *Session) {
	dav.locker.Lock()
	defer dav.locker.Unlock()
	m, ok := dav.connections[s.Host]
	if !ok || m == nil {
		return
	}
	m[s.Workdir] = nil
	return
}

func (dav *WebdavHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer err2.Catch(func(err error) {
		w.WriteHeader(500)
		fmt.Fprintln(w, err)
	})

	host := try.To1(getHost(dav.HostMatcher, r.Host))
	session := dav.getSession(host, r.URL.Path)
	if session == nil {
		w.WriteHeader(403)
		fmt.Fprintf(w, "no webdav server for this host %s", host)
		return
	}

	session.ServeHTTP(w, r)
}

func getHost(matcher *regexp.Regexp, host string) (h string, err error) {
	defer err2.Handle(&err)
	matched := matcher.FindStringSubmatch(host)
	if len(matched) < 2 {
		err2.Throwf("match host failed")
	}
	return matched[1], nil
}

func (dav *WebdavHub) getSession(host, path string) (s *Session) {
	dav.locker.RLock()
	defer dav.locker.RUnlock()
	m, ok := dav.connections[host]
	if !ok {
		return
	}
	var cv current[*Session]
	for k, v := range m {
		d := strings.TrimPrefix(path, k)
		if len(d) == len(path) { //没有匹配
			continue
		}
		// 匹配越多越好
		nv := current[*Session]{
			point: len(path) - len(d),
			v:     v,
		}
		if nv.point > cv.point {
			cv = nv
		}
	}
	return cv.v
}

type current[V any] struct {
	point int
	v     V
}
