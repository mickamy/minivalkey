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

func TestServer_cmdTTL(t *testing.T) {
	t.Parallel()

	base := time.Unix(0, 0)

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*db.DB)
		want    string
	}{
		{
			name: "returns ttl when key has expiry",
			args: resp.Args{
				[]byte("ttl"),
				[]byte("foo"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "bar", base.Add(5*time.Second))
			},
			want: ":5\r\n",
		},
		{
			name: "returns minus one when key has no expiry",
			args: resp.Args{
				[]byte("ttl"),
				[]byte("foo"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "bar", time.Time{})
			},
			want: ":-1\r\n",
		},
		{
			name: "returns minus two when key is missing",
			args: resp.Args{
				[]byte("ttl"),
				[]byte("missing"),
			},
			want: ":-2\r\n",
		},
		{
			name: "complains when key argument is missing",
			args: resp.Args{
				[]byte("ttl"),
			},
			want: "-ERR wrong number of arguments for 'ttl' command\r\n",
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
				clock: clock.New(base),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))
			req := newRequest(session.New(), "TTL", tc.args)

			if err := srv.cmdTTL(w, req); err != nil {
				t.Fatalf("cmdTTL returned error: %v", err)
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
