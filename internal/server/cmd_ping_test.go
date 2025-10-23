package server

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/session"
)

func TestServer_cmdPing(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tcs := []struct {
		name string
		args resp.Args
		want string
	}{
		{
			name: "returns pong when no message is given",
			args: resp.Args{
				[]byte("ping"),
			},
			want: "+PONG\r\n",
		},
		{
			name: "echoes payload when provided",
			args: resp.Args{
				[]byte("ping"),
				[]byte("hello"),
			},
			want: "$5\r\nhello\r\n",
		},
		{
			name: "complains when too many arguments are provided",
			args: resp.Args{
				[]byte("ping"),
				[]byte("one"),
				[]byte("two"),
			},
			want: "-ERR wrong number of arguments for 'ping' command\r\n",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := &Server{
				dbMap: make(map[int]*db.DB),
				clock: clock.New(now),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))
			req := newRequest(session.New(), "PING", tc.args)

			if err := srv.cmdPing(w, req); err != nil {
				t.Fatalf("cmdPing returned error: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", tc.want, got)
			}
		})
	}
}
