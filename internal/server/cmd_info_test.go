package server

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/store"
)

func TestServer_cmdInfo(t *testing.T) {
	t.Parallel()

	base := time.Unix(1_000, 0)

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*store.Store)
		want    string
		wantFn  func(*store.Store, *Server) string
	}{
		{
			name: "returns default section when none is provided",
			args: resp.Args{
				[]byte("info"),
			},
			wantFn: func(st *store.Store, srv *Server) string {
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
			arrange: func(st *store.Store) {
				st.SetString("foo", "bar", time.Time{})
				st.SetString("baz", "qux", base.Add(30*time.Second))
			},
			wantFn: func(st *store.Store, srv *Server) string {
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

			st := store.New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			srv := &Server{
				store: st,
				clock: clock.New(base),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))

			if err := srv.cmdInfo("INFO", tc.args, w); err != nil {
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
