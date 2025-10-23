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

func TestServer_cmdSet(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*db.DB)
		assert  func(*testing.T, *db.DB)
		want    string
	}{
		{
			name: "stores value and replies with ok",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok {
					t.Fatalf("foo missing from db")
				}
				if got != "bar" {
					t.Fatalf("expected value bar, got %q", got)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "complains when value is missing",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
			},
			want: "-ERR wrong number of arguments for 'set' command\r\n",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := db.New()
			if tc.arrange != nil {
				tc.arrange(d)
			}
			srv := &Server{
				dbMap: map[int]*db.DB{0: d},
				clock: clock.New(now),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))
			req := newRequest(session.New(), "SET", tc.args)

			if err := srv.cmdSet(w, req); err != nil {
				t.Fatalf("cmdSet returned error: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", tc.want, got)
			}
			if tc.assert != nil {
				tc.assert(t, d)
			}
		})
	}
}
