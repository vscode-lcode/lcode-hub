package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/alessio/shellescape"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
	"github.com/vscode-lcode/bash"
	"github.com/vscode-lcode/bash/server"
	"github.com/vscode-lcode/bash/webdav"
)

var args struct {
	addr        string
	hello       string
	hostMacther string
	timeout     time.Duration
}

var Version = "dev"
var f = flag.NewFlagSet("lcode-hub@"+Version, flag.ExitOnError)

var defualtPort = 4349
var defaultHostMatcher = `^(.+)\.lo`
var defaultTimeout = 5 * time.Second

var dev = false

func init() {
	if Version == "dev" {
		defualtPort = 43499
		dev = true
	}
	f.StringVar(&args.addr, "addr", fmt.Sprintf("127.0.0.1:%d", defualtPort), "local-hub listen addr")
	f.StringVar(&args.hello, "hello", fmt.Sprintf("webdav://{{.host}}.lo.shynome.com:%d{{.path}}", defualtPort), "")
	f.StringVar(&args.hostMacther, "host-finder", defaultHostMatcher, "host regexp matcher")
	f.DurationVar(&args.timeout, "timeout", defaultTimeout, "timeout")
}

var subShellTpl = func() (tpl string) {
	tpl = `exec <>/dev/tcp/{{.server}} 2> >(%s) bash +o history -i -s -- -leader=false {{.args}}`
	tpl = fmt.Sprintf(tpl, `4>&0 5>/dev/tcp/{{.server}} 3> >(>&5 echo '>2: {{.head}}' && >&5 dd conv=sync <&4) dd conv=sync <&5 >&2`)
	return tpl
}()

func main() {
	f.Parse(os.Args[1:])

	if err := hasRunning(args.addr); err == nil {
		return
	} else {
		logger.Println("got version failed.", err)
	}

	l := try.To1(net.Listen("tcp", args.addr))
	port := func() uint16 {
		ap := netip.MustParseAddrPort(l.Addr().String())
		return ap.Port()
	}()

	var webdavLink *template.Template
	{
		webdavLinkTpl := fmt.Sprintf("webdav://{{.host}}.lo.shynome.com:%d{{.path}}", port)
		webdavLink = try.To1(template.New("webdav").Parse(webdavLinkTpl))
	}

	hello := try.To1(template.New("hello").Parse(args.hello))
	subShell := try.To1(template.New("subshell").Parse(subShellTpl))
	_ = subShell

	hostMatcher := regexp.MustCompile(args.hostMacther)

	whub := NewWebdavHub()
	whub.HostMatcher = hostMatcher

	sup := NewSupervisor()
	sup.HostMatcher = hostMatcher

	hub := server.NewHub()
	hub.Timeout = args.timeout
	hub.OnSessionOpen = func(c bash.Session) func() {
		defer err2.Catch(func(err error) {
			fmt.Println("err", err)
			c.Close()
		})

		var args struct {
			leader bool
			addr   string
			raw    string
			rest   []string
		}
		{
			f := flag.NewFlagSet("lcode@"+Version, flag.ContinueOnError)
			f.BoolVar(&args.leader, "leader", true, "")
			f.StringVar(&args.addr, "server", fmt.Sprintf("127.0.0.1/%d", defualtPort), "")
			args.raw = string(try.To1(c.Run("echo -n $@")))

			var output bytes.Buffer
			f.SetOutput(&output)

			argsArr := strings.Split(args.raw, " ")
			err := f.Parse(argsArr)
			if err != nil {
				s := output.String()
				s = shellescape.Quote(s)
				cmd := fmt.Sprintf(" >&2 echo %s", s)
				fmt.Fprintln(c.(*bash.Client).Conn, cmd)
				err2.Throwf("args parse failed")
			}
			args.rest = f.Args()
		}
		if args.leader {
			h := sup.NewHandler()
			if dev {
				logger.Println("start leader", h.String())
			}
			child := try.To1(ExecTpl(subShell, map[string]string{
				"server": args.addr,
				"args":   args.raw,
				"head":   h.String(),
			}))
			// 启动子进程处理命令, 便于设置复杂的命令行选项
			fmt.Fprintln(c.(*bash.Client).Conn, fmt.Sprintf(" %s", child))
			go func() {
				defer c.Close()
				select {
				case <-time.After(hub.Timeout):
					logger.Println("child start timeout.")
				case stderr := <-h.ch:
					<-stderr.closed
					s := whub.getSession(stderr.Host, stderr.Workdir)
					if s != nil {
						s.Session.Close()
					}
					logger.Println("child has been closed")
				}
			}()

			return func() {
				logger.Println("leader closed", h.String())
			}
		}

		if dev {
			logger.Println("main start")
		}

		if len(args.rest) == 0 {
			args.rest = []string{"."}
		}

		// 清除 PS1, 便于解析 stderr
		fmt.Fprintln(c.(*bash.Client).Conn, `export PS1=""`)

		f := webdav.OpenFile(c, "/proc/sys/kernel/hostname")
		host := string(try.To1(io.ReadAll(f)))
		f.Close()
		host = strings.TrimSuffix(host, "\n")
		host = strings.ToLower(host)

		pwd := string(try.To1(c.Run("pwd")))
		pwd = strings.TrimSuffix(pwd, "\n")

		s, err := whub.NewSession(host, pwd, c)
		if err != nil {
			s := ">2: " + err.Error()
			try.To1(c.Run(">&2 echo " + shellescape.Quote(s)))
			try.To(err)
		}

		welcome := try.To1(ExecTpl(webdavLink, map[string]string{"host": host, "path": pwd + "/"}))
		welcome = ">2: " + welcome
		try.To1(c.Run(">&2 echo " + shellescape.Quote(welcome)))

		go func() {
			defer err2.Catch(func(err error) {
				warn := fmt.Sprintf(">2: err: %s", err.Error())
				c.Run(">&2 echo " + shellescape.Quote(warn))
				c.Close()
			})
			var hit = 0
			for _, path := range args.rest {
				path = filepath.Join(pwd, path)
				f := webdav.OpenFile(c, path)
				finfo, err := f.Stat()
				if errors.Is(err, os.ErrNotExist) {
					tip := fmt.Sprintf(">2: 404://%s path not exist.", path)
					try.To1(c.Run(">&2 echo " + shellescape.Quote(tip)))
					continue
				}
				z := try.To1(filepath.Rel(pwd, path))
				if strings.HasPrefix(z, "..") {
					tip := fmt.Sprintf(">2: 404://%s don't allow edit parent directory files", path)
					try.To1(c.Run(">&2 echo " + shellescape.Quote(tip)))
					continue
				}

				hit++
				if finfo.IsDir() {
					path += "/"
				}
				editLink := try.To1(ExecTpl(hello, map[string]string{"host": host, "path": path}))
				editLink = ">2: " + editLink
				if editLink == welcome {
					continue
				}
				try.To1(c.Run(">&2 echo " + shellescape.Quote(editLink)))
			}
			if hit == 0 {
				err2.Throwf("no edit target. exitted.")
			}
		}()

		if dev {
			logger.Println("main started.", s.String())
		}

		return func() {
			if dev {
				logger.Println("remove main", s.String())
			}
			whub.RemoveSession(s)
		}
	}

	// 端口复用
	sl := SwitchListener(l)
	sl.Switch = func(line string, conn net.Conn) (gohttp bool) {
		switch {
		case strings.HasPrefix(line, ">2"):
			go sup.StderrPipe(conn)
			return false
		case strings.HasPrefix(line, "0"):
			go hub.ServeConn(conn)
			return false
		}
		return true
	}

	// webdav server
	logger.Println(f.Name(), "is running on", l.Addr().String())
	try.To(http.Serve(sl, whub))
}

func ExecTpl(tpl *template.Template, data any) (s string, err error) {
	var b bytes.Buffer
	if err = tpl.Execute(&b, data); err != nil {
		return
	}
	return b.String(), nil
}

type ConnWithReader struct {
	io.Reader
	net.Conn
}

func (conn *ConnWithReader) Read(p []byte) (n int, err error) {
	return conn.Reader.Read(p)
}
