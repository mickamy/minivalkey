package resp_test

import (
	"bufio"
	"bytes"
	"errors"
	"testing"

	"github.com/mickamy/minivalkey/internal/resp"
)

func TestWriter_WriteCommands(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name  string
		write func(*resp.Writer) error
		want  string
	}{
		{
			name: "simple string",
			write: func(w *resp.Writer) error {
				return w.WriteString("PONG")
			},
			want: "+PONG\r\n",
		},
		{
			name: "errors",
			write: func(w *resp.Writer) error {
				if err := w.WriteErrorString("ERR failure"); err != nil {
					return err
				}
				return w.WriteError(errors.New("ERR boom"))
			},
			want: "-ERR failure\r\n-ERR boom\r\n",
		},
		{
			name: "error and flush",
			write: func(w *resp.Writer) error {
				return w.WriteErrorAndFlush(errors.New("ERR flushed"))
			},
			want: "-ERR flushed\r\n",
		},
		{
			name: "integers",
			write: func(w *resp.Writer) error {
				return w.WriteInt(42)
			},
			want: ":42\r\n",
		},
		{
			name: "bulk strings",
			write: func(w *resp.Writer) error {
				if err := w.WriteBulk([]byte("foo")); err != nil {
					return err
				}
				return w.WriteBulk(nil)
			},
			want: "$3\r\nfoo\r\n$-1\r\n",
		},
		{
			name: "arrays",
			write: func(w *resp.Writer) error {
				if err := w.WriteArrayHeader(3); err != nil {
					return err
				}
				if err := w.WriteBulkElem([]byte("SET")); err != nil {
					return err
				}
				if err := w.WriteBulkElem(nil); err != nil {
					return err
				}
				return w.WriteIntElem(10)
			},
			want: "*3\r\n$3\r\nSET\r\n$-1\r\n:10\r\n",
		},
		{
			name: "empty array and null",
			write: func(w *resp.Writer) error {
				if err := w.WriteEmptyArray(); err != nil {
					return err
				}
				return w.WriteNull()
			},
			want: "*0\r\n$-1\r\n",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := new(bytes.Buffer)
			w := resp.NewWriter(bufio.NewWriter(buf))

			if err := tc.write(w); err != nil {
				t.Fatalf("write failed: %v", err)
			}
			if err := w.Flush(); err != nil {
				t.Fatalf("flush failed: %v", err)
			}
			got := buf.String()
			if got != tc.want {
				t.Fatalf("unexpected payload:\nwant %q\ngot  %q", tc.want, got)
			}
		})
	}
}
