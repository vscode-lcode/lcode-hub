package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net"
	"time"

	"github.com/vscode-lcode/bash/server"
)

type HubFirstListener struct {
	net.Listener

	Switch func(line string, conn net.Conn) (gohttp bool)
}

var _ net.Listener = (*HubFirstListener)(nil)

func SwitchListener(l net.Listener) *HubFirstListener {
	return &HubFirstListener{
		Listener: l,
	}
}

func (sw *HubFirstListener) Accept() (net.Conn, error) {
	w := make(chan Item)
	go func() {
		for {
			conn, err := sw.Listener.Accept()
			if err != nil {
				w <- Item{nil, err}
				break
			}
			got := func(conn net.Conn) (got bool) {
				var firstLine string
				var r io.Reader

				ctx := context.Background()
				ctx, cause := context.WithCancelCause(ctx)
				time.AfterFunc(1*time.Second, func() {
					defer cause(context.Canceled)
					if err := ctx.Err(); err != nil {
						return
					}
					var h server.Header
					h[1] = byte(server.MsgInitSession)

					firstLine = hex.EncodeToString(h[:])
					r = io.MultiReader(
						bytes.NewBufferString(firstLine),
						conn,
					)
				})
				go func() {
					defer cause(context.Canceled)
					rr := bufio.NewReader(conn)
					line, _, err := rr.ReadLine()
					if err != nil {
						cause(err)
						return
					}

					firstLine = string(line)
					r = io.MultiReader(
						bytes.NewReader(line),
						bytes.NewReader([]byte("\n")),
						rr,
					)
				}()

				<-ctx.Done()

				if err := context.Cause(ctx); err != context.Canceled {
					conn.Close()
					return
				}

				conn = &ConnWithReader{
					Reader: r,
					Conn:   conn,
				}

				if sw.Switch != nil {
					gonext := sw.Switch(firstLine, conn)
					if !gonext {
						return
					}
				}

				w <- Item{conn, nil}
				return true
			}(conn)
			if got {
				return
			}
		}
	}()
	item := <-w
	return item.conn, item.err
}

type Item struct {
	conn net.Conn
	err  error
}
