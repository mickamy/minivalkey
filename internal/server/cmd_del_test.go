package server

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
)

func TestServer_cmdDel(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*db.DB)
		assert  func(*testing.T, *db.DB)
		want    string
	}{
		{
			name: "returns deleted count for existing keys",
			args: resp.Args{
				[]byte("del"),
				[]byte("foo"),
				[]byte("bar"),
			},
			arrange: func(st *db.DB) {
				st.SetString("foo", "1", time.Time{})
				st.SetString("bar", "2", time.Time{})
			},
			assert: func(t *testing.T, st *db.DB) {
				if _, ok := st.GetString(time.Time{}, "foo"); ok {
					t.Fatalf("foo was not deleted")
				}
				if _, ok := st.GetString(time.Time{}, "bar"); ok {
					t.Fatalf("bar was not deleted")
				}
			},
			want: ":2\r\n",
		},
		{
			name: "returns zero when keys do not exist",
			args: resp.Args{
				[]byte("del"),
				[]byte("missing"),
			},
			want: ":0\r\n",
		},
		{
			name: "complains when no keys are provided",
			args: resp.Args{
				[]byte("del"),
			},
			want: "-ERR wrong number of arguments for 'del' command\r\n",
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
			srv := &Server{db: st}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))

			if err := srv.cmdDel("DEL", tc.args, w); err != nil {
				t.Fatalf("cmdDel returned error: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", tc.want, got)
			}
			if tc.assert != nil {
				tc.assert(t, st)
			}
		})
	}
}
