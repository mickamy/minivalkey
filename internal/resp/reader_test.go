package resp_test

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mickamy/minivalkey/internal/resp"
)

func TestReader_ReadArrayBulk(t *testing.T) {
	t.Parallel()

	bulk := func(s string) resp.Arg { return []byte(s) }

	tcs := []struct {
		name            string
		payload         string
		want            resp.Args
		wantErr         error
		wantErrContains string
	}{
		{
			name:    "simple",
			payload: "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
			want: resp.Args{
				bulk("GET"),
				bulk("key"),
			},
		},
		{
			name:    "with null bulk",
			payload: "*3\r\n$4\r\nLLEN\r\n$-1\r\n$1\r\nx\r\n",
			want: resp.Args{
				bulk("LLEN"),
				nil,
				bulk("x"),
			},
		},
		{
			name:            "wrong prefix",
			payload:         "$3\r\nfoo\r\n",
			wantErr:         resp.ErrProtocol,
			wantErrContains: "expected array",
		},
		{
			name:    "negative array size",
			payload: "*1\r\n$-2\r\n",
			wantErr: resp.ErrProtocol,
		},
		{
			name:    "short bulk data",
			payload: "*1\r\n$4\r\nfoo",
			wantErr: io.ErrUnexpectedEOF,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reader := resp.NewReader(bufio.NewReader(bytes.NewBufferString(tc.payload)))
			got, err := reader.ReadArrayBulk()

			if tc.wantErr != nil {
				if err == nil || !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				if tc.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrContains)) {
					t.Fatalf("expected error to contain %q, got %v", tc.wantErrContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d args, got %d", len(tc.want), len(got))
			}
			for i := range got {
				if got[i] == nil || tc.want[i] == nil {
					if !(got[i] == nil && tc.want[i] == nil) {
						t.Fatalf("arg[%d] mismatch: expected %v, got %v", i, tc.want[i], got[i])
					}
					continue
				}
				if !bytes.Equal(got[i], tc.want[i]) {
					t.Fatalf("arg[%d] mismatch: expected %q, got %q", i, tc.want[i], got[i])
				}
			}
		})
	}
}
