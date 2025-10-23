package server

import (
	"bufio"
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/session"
)

func TestServer_cmdSet(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_000, 0)

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
			name: "sets expire with EX",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("EX"),
				[]byte("10"),
			},
			assert: func(t *testing.T, db *db.DB) {
				if ttl := db.TTL(now, "foo"); ttl != 10 {
					t.Fatalf("expected ttl 10, got %d", ttl)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "sets expire with PX",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("px"),
				[]byte("1000"),
			},
			assert: func(t *testing.T, db *db.DB) {
				if ttl := db.TTL(now, "foo"); ttl != 1 {
					t.Fatalf("expected ttl 1, got %d", ttl)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "sets expire with EXAT",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("EXAT"),
				[]byte(strconv.FormatInt(now.Add(15*time.Second).Unix(), 10)),
			},
			assert: func(t *testing.T, db *db.DB) {
				if ttl := db.TTL(now, "foo"); ttl != 15 {
					t.Fatalf("expected ttl 15, got %d", ttl)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "sets expire with PXAT",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("pxat"),
				[]byte(strconv.FormatInt(now.Add(1500*time.Millisecond).UnixMilli(), 10)),
			},
			assert: func(t *testing.T, db *db.DB) {
				if ttl := db.TTL(now, "foo"); ttl != 1 {
					t.Fatalf("expected ttl 1, got %d", ttl)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "rejects invalid expire time",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("EX"),
				[]byte("0"),
			},
			want: "-ERR invalid expire time in set\r\n",
		},
		{
			name: "rejects missing expire argument",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("EX"),
			},
			want: "-ERR syntax error\r\n",
		},
		{
			name: "rejects non integer expire argument",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("PX"),
				[]byte("abc"),
			},
			want: "-ERR value is not an integer or out of range\r\n",
		},
		{
			name: "stores only when NX condition met",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("fresh"),
				[]byte("NX"),
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "fresh" {
					t.Fatalf("expected foo=fresh, got %q ok=%v", got, ok)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "returns old value with GET option",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("baz"),
				[]byte("GET"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "bar", time.Time{})
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "baz" {
					t.Fatalf("expected foo=baz, got %q ok=%v", got, ok)
				}
			},
			want: "$3\r\nbar\r\n",
		},
		{
			name: "returns null with GET when key missing",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("baz"),
				[]byte("GET"),
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "baz" {
					t.Fatalf("expected foo=baz, got %q ok=%v", got, ok)
				}
			},
			want: "$-1\r\n",
		},
		{
			name: "returns null with GET when NX succeeds on fresh key",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("baz"),
				[]byte("NX"),
				[]byte("GET"),
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "baz" {
					t.Fatalf("expected foo=baz, got %q ok=%v", got, ok)
				}
			},
			want: "$-1\r\n",
		},
		{
			name: "returns null with GET when NX condition fails",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("new"),
				[]byte("NX"),
				[]byte("GET"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "old", time.Time{})
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "old" {
					t.Fatalf("expected foo to remain old, got %q ok=%v", got, ok)
				}
			},
			want: "$-1\r\n",
		},
		{
			name: "rejects duplicate GET option",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("GET"),
				[]byte("GET"),
			},
			want: "-ERR syntax error\r\n",
		},
		{
			name: "returns null when NX condition fails",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("new"),
				[]byte("NX"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "old", time.Time{})
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "old" {
					t.Fatalf("expected foo to remain old, got %q ok=%v", got, ok)
				}
			},
			want: "$-1\r\n",
		},
		{
			name: "stores only when XX condition met",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("new"),
				[]byte("XX"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "old", time.Time{})
			},
			assert: func(t *testing.T, db *db.DB) {
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "new" {
					t.Fatalf("expected foo=new, got %q ok=%v", got, ok)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "returns null when XX condition fails",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("new"),
				[]byte("XX"),
			},
			want: "$-1\r\n",
		},
		{
			name: "keeps existing ttl with keepttl",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("updated"),
				[]byte("KEEPTTL"),
			},
			arrange: func(db *db.DB) {
				db.SetString("foo", "old", now.Add(10*time.Second))
			},
			assert: func(t *testing.T, db *db.DB) {
				if ttl := db.TTL(now, "foo"); ttl != 10 {
					t.Fatalf("expected ttl 10, got %d", ttl)
				}
				got, ok := db.GetString(time.Time{}, "foo")
				if !ok || got != "updated" {
					t.Fatalf("expected foo=updated, got %q ok=%v", got, ok)
				}
			},
			want: "+OK\r\n",
		},
		{
			name: "rejects conflicting nx and xx",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("NX"),
				[]byte("XX"),
			},
			want: "-ERR syntax error\r\n",
		},
		{
			name: "rejects keepttl combined with expiration option",
			args: resp.Args{
				[]byte("set"),
				[]byte("foo"),
				[]byte("bar"),
				[]byte("keepttl"),
				[]byte("EX"),
				[]byte("10"),
			},
			want: "-ERR syntax error\r\n",
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
