package server

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/session"
)

func TestServer_cmdInfo(t *testing.T) {
	t.Parallel()

	base := time.Unix(1_000, 0)

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*db.DB)
		want    string
		wantFn  func(*db.DB, *Server) string
	}{
		{
			name: "returns default section when none is provided",
			args: resp.Args{
				[]byte("info"),
			},
			wantFn: func(st *db.DB, srv *Server) string {
				now := srv.Now()
				txt, _ := buildInfo("default", now, st, srv.uptimeSeconds(now))
				return fmt.Sprintf("$%d\r\n%s\r\n", len(txt), txt)
			},
		},
		{
			name: "returns memory section when requested",
			args: resp.Args{
				[]byte("info"),
				[]byte("memory"),
			},
			arrange: func(st *db.DB) {
				st.SetString("foo", "bar", time.Time{})
				st.SetString("baz", "qux", base.Add(30*time.Second))
			},
			wantFn: func(st *db.DB, srv *Server) string {
				now := srv.Now()
				txt, _ := buildInfo("memory", now, st, srv.uptimeSeconds(now))
				return fmt.Sprintf("$%d\r\n%s\r\n", len(txt), txt)
			},
		},
		{
			name: "complains when section is unknown",
			args: resp.Args{
				[]byte("info"),
				[]byte("mystery"),
			},
			want: "-ERR unknown section\r\n",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := db.New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			srv := &Server{
				clock: clock.New(base),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))
			req := newRequest(session.New(), "INFO", tc.args)

			if err := srv.cmdInfo(w, req); err != nil {
				t.Fatalf("cmdInfo returned error: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}

			want := tc.want
			if tc.wantFn != nil {
				want = tc.wantFn(st, srv)
			}
			if got := buf.String(); got != want {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", want, got)
			}
		})
	}
}
