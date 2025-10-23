package server

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/session"
)

func TestServer_cmdHello(t *testing.T) {
	t.Parallel()

	now := time.Now()

	const wantHello = "" +
		"*14\r\n" +
		"$6\r\nserver\r\n" +
		"$6\r\nvalkey\r\n" +
		"$7\r\nversion\r\n" +
		"$5\r\n0.0.0\r\n" +
		"$5\r\nproto\r\n" +
		":2\r\n" +
		"$2\r\nid\r\n" +
		":1\r\n" +
		"$4\r\nmode\r\n" +
		"$10\r\nstandalone\r\n" +
		"$4\r\nrole\r\n" +
		"$6\r\nmaster\r\n" +
		"$7\r\nmodules\r\n" +
		"*0\r\n"

	tcs := []struct {
		name string
		args resp.Args
	}{
		{
			name: "returns handshake without arguments",
			args: resp.Args{
				[]byte("hello"),
			},
		},
		{
			name: "returns handshake when proto is requested",
			args: resp.Args{
				[]byte("hello"),
				[]byte("3"),
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &Server{
				clock: clock.New(now),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))
			req := newRequest(session.New(), "HELLO", tc.args)

			if err := srv.cmdHello(w, req); err != nil {
				t.Fatalf("cmdHello returned error: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}
			if got := buf.String(); got != wantHello {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", wantHello, got)
			}
		})
	}
}
