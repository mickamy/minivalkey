package resp_test

import (
	"testing"

	"github.com/mickamy/minivalkey/internal/resp"
)

func TestParseInt(t *testing.T) {
	t.Parallel()

	tcs := []struct {
		name   string
		input  []byte
		want   int64
		wantOK bool
	}{
		{
			name:   "simple positive",
			input:  []byte("123"),
			want:   123,
			wantOK: true,
		},
		{
			name:   "simple negative",
			input:  []byte("-42"),
			want:   -42,
			wantOK: true,
		},
		{
			name:   "zero",
			input:  []byte("0"),
			want:   0,
			wantOK: true,
		},
		{
			name:   "leading zeros",
			input:  []byte("001"),
			want:   1,
			wantOK: true,
		},
		{
			name:   "non digit",
			input:  []byte("abc"),
			wantOK: false,
		},
		{
			name:   "negative non digit",
			input:  []byte("-x"),
			wantOK: false,
		},
		{
			name:   "empty",
			input:  []byte(""),
			wantOK: false,
		},
		{
			name:   "plus sign unsupported",
			input:  []byte("+1"),
			wantOK: false,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := resp.ParseInt(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ParseInt(%q) ok = %v; want %v", tc.input, ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got != tc.want {
				t.Fatalf("ParseInt(%q) = %d; want %d", tc.input, got, tc.want)
			}
		})
	}
}
