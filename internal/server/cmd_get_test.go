package server

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
)

func TestServer_cmdGet(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*db.DB)
		want    string
	}{
		{
			name: "returns stored value",
			args: resp.Args{
				[]byte("get"),
				[]byte("foo"),
			},
			arrange: func(st *db.DB) {
				st.SetString("foo", "bar", time.Time{})
			},
			want: "$3\r\nbar\r\n",
		},
		{
			name: "returns null for missing value",
			args: resp.Args{
				[]byte("get"),
				[]byte("missing"),
			},
			want: "$-1\r\n",
		},
		{
			name: "complains when key is missing",
			args: resp.Args{
				[]byte("get"),
			},
			want: "-ERR wrong number of arguments for 'get' command\r\n",
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
				db:    st,
				clock: clock.New(now),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))

			if err := srv.cmdGet("GET", tc.args, w); err != nil {
				t.Fatalf("cmdGet returned error: %v", err)
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
