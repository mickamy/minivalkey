package server

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/store"
)

func TestServer_cmdSet(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*store.Store)
		assert  func(*testing.T, *store.Store)
		want    string
	}{
		{
			name: "stores value and replies with ok",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
			},
			assert: func(t *testing.T, st *store.Store) {
				got, ok := st.GetString(time.Time{}, "foo")
				if !ok {
					t.Fatalf("foo missing from store")
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

			st := store.New()
			if tc.arrange != nil {
				tc.arrange(st)
			}
			srv := &Server{
				store: st,
				clock: clock.New(now),
			}

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))

			if err := srv.cmdSet("SET", tc.args, w); err != nil {
				t.Fatalf("cmdSet returned error: %v", err)
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
