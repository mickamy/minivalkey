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

func TestServer_cmdExists(t *testing.T) {
	t.Parallel()

	base := time.Unix(0, 0)

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*db.DB, time.Time)
		want    string
	}{
		{
			name: "counts existing keys",
			args: resp.Args{
				[]byte("exists"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("baz"),
			},
			arrange: func(db *db.DB, now time.Time) {
				db.SetString("foo", "1", time.Time{})
				db.SetString("bar", "2", time.Time{})
			},
			want: ":2\r\n",
		},
		{
			name: "ignores expired keys",
			args: resp.Args{
				[]byte("exists"),
				[]byte("fresh"),
				[]byte("stale"),
			},
			arrange: func(db *db.DB, now time.Time) {
				db.SetString("fresh", "1", time.Time{})
				db.SetString("stale", "2", now.Add(-time.Second))
			},
			want: ":1\r\n",
		},
		{
			name: "complains when no keys are provided",
			args: resp.Args{
				[]byte("exists"),
			},
			want: "-ERR wrong number of arguments for 'exists' command\r\n",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := db.New()
			if tc.arrange != nil {
				tc.arrange(d, base)
			}
			srv := &Server{
				dbMap: map[int]*db.DB{0: d},
				clock: clock.New(base),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))
			req := newRequest(session.New(), "EXISTS", tc.args)

			if err := srv.cmdExists(w, req); err != nil {
				t.Fatalf("cmdExists returned error: %v", err)
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
