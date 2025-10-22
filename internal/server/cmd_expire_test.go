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

func TestServer_cmdExpire(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tcs := []struct {
		name    string
		args    resp.Args
		arrange func(*store.Store)
		assert  func(*testing.T, *store.Store, *Server)
		want    string
	}{
		{
			name: "sets ttl when key exists",
			args: resp.Args{
				[]byte("expire"),
				[]byte("foo"),
				[]byte("10"),
			},
			arrange: func(st *store.Store) {
				st.SetString("foo", "bar", time.Time{})
			},
			assert: func(t *testing.T, st *store.Store, srv *Server) {
				if ttl := st.TTL(srv.Now(), "foo"); ttl != 10 {
					t.Fatalf("expected ttl 10, got %d", ttl)
				}
			},
			want: ":1\r\n",
		},
		{
			name: "removes ttl when negative seconds are given",
			args: resp.Args{
				[]byte("expire"),
				[]byte("foo"),
				[]byte("-1"),
			},
			arrange: func(st *store.Store) {
				st.SetString("foo", "bar", now.Add(10*time.Second))
				st.Expire(now, "foo", 10)
			},
			assert: func(t *testing.T, st *store.Store, srv *Server) {
				if ttl := st.TTL(srv.Now(), "foo"); ttl != -1 {
					t.Fatalf("expected ttl -1, got %d", ttl)
				}
			},
			want: ":1\r\n",
		},
		{
			name: "returns zero when key does not exist",
			args: resp.Args{
				[]byte("expire"),
				[]byte("missing"),
				[]byte("12"),
			},
			want: ":0\r\n",
		},
		{
			name: "complains when ttl is not an integer",
			args: resp.Args{
				[]byte("expire"),
				[]byte("foo"),
				[]byte("abc"),
			},
			want: "-ERR value is not an integer or out of range\r\n",
		},
		{
			name: "complains when ttl argument is missing",
			args: resp.Args{
				[]byte("expire"),
				[]byte("foo"),
			},
			want: "-ERR wrong number of arguments for 'expire' command\r\n",
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

			if err := srv.cmdExpire("EXPIRE", tc.args, w); err != nil {
				t.Fatalf("cmdExpire returned error: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", tc.want, got)
			}
			if tc.assert != nil {
				tc.assert(t, st, srv)
			}
		})
	}
}
