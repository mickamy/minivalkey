package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var (
	// ErrProtocol indicates a malformed RESP2 payload.
	ErrProtocol = errors.New("resp: protocol error")
)

// Reader provides RESP2 read helpers over a buffered reader.
type Reader struct {
	r *bufio.Reader
}

// NewReader wraps the provided bufio.Reader.
func NewReader(r *bufio.Reader) *Reader {
	return &Reader{r: r}
}

// ReadArrayBulk reads a RESP2 array of bulk strings.
// e.g. *3\r\n$3\r\nSET\r\n$1\r\na\r\n$1\r\nb\r\n
func (r *Reader) ReadArrayBulk() (Args, error) {
	prefix, err := r.r.ReadByte()
	if err != nil {
		return nil, err
	}
	if prefix != '*' {
		return nil, fmt.Errorf("%w: expected array '*', got %q", ErrProtocol, prefix)
	}
	n, err := r.readIntegerLine()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, ErrProtocol
	}
	out := make(Args, 0, n)
	for i := 0; i < n; i++ {
		pfx, err := r.r.ReadByte()
		if err != nil {
			return nil, err
		}
		if pfx != '$' {
			return nil, fmt.Errorf("%w: expected bulk '$', got %q", ErrProtocol, pfx)
		}
		size, err := r.readIntegerLine()
		if err != nil {
			return nil, err
		}
		if size == -1 {
			out = append(out, nil)
			continue
		}
		if size < 0 {
			return nil, ErrProtocol
		}
		buf := make([]byte, size)
		if _, err := io.ReadFull(r.r, buf); err != nil {
			return nil, err
		}
		if err := r.expectCRLF(); err != nil {
			return nil, err
		}
		out = append(out, buf)
	}
	return out, nil
}

func (r *Reader) readIntegerLine() (int, error) {
	line, err := r.r.ReadString('\n')
	if err != nil {
		return 0, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return 0, fmt.Errorf("%w: expected '$', got %q", ErrProtocol, line)
	}
	line = line[:len(line)-2]
	n, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("%w: expected integer, got %q", ErrProtocol, line)
	}
	return n, nil
}

func (r *Reader) expectCRLF() error {
	b1, err := r.r.ReadByte()
	if err != nil {
		return err
	}
	b2, err := r.r.ReadByte()
	if err != nil {
		return err
	}
	if b1 != '\r' || b2 != '\n' {
		return fmt.Errorf("%w: expected CRLF, got %q%q", ErrProtocol, b1, b2)
	}
	return nil
}
