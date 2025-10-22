package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// ------ Writer (server -> client) for RESP2 types ------

func WriteSimpleString(w *bufio.Writer, s string) error {
	if _, err := w.WriteString("+" + s + "\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

func WriteError(w *bufio.Writer, msg string) error {
	if _, err := w.WriteString("-" + msg + "\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

func WriteInt(w *bufio.Writer, n int64) error {
	if _, err := w.WriteString(":" + strconv.FormatInt(n, 10) + "\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

func WriteBulk(w *bufio.Writer, b []byte) error {
	if b == nil {
		// Null Bulk String
		if _, err := w.WriteString("$-1\r\n"); err != nil {
			return err
		}
		return w.Flush()
	}
	if _, err := w.WriteString("$" + strconv.Itoa(len(b)) + "\r\n"); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.WriteString("\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

func WriteArrayHeader(w *bufio.Writer, n int) error {
	if _, err := w.WriteString("*" + strconv.Itoa(n) + "\r\n"); err != nil {
		return err
	}
	return nil
}

func WriteBulkElem(w *bufio.Writer, b []byte) error {
	if b == nil {
		if _, err := w.WriteString("$-1\r\n"); err != nil {
			return err
		}
		return nil
	}
	if _, err := w.WriteString("$" + strconv.Itoa(len(b)) + "\r\n"); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	_, err := w.WriteString("\r\n")
	return err
}

func WriteIntElem(w *bufio.Writer, n int64) error {
	if _, err := w.WriteString(":" + strconv.FormatInt(n, 10) + "\r\n"); err != nil {
		return err
	}
	return nil
}

func WriteEmptyArray(w *bufio.Writer) error {
	if _, err := w.WriteString("*0\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

func WriteNull(w *bufio.Writer) error {
	// RESP2 Null Bulk String
	if _, err := w.WriteString("$-1\r\n"); err != nil {
		return err
	}
	return w.Flush()
}

func Flush(w *bufio.Writer) error { return w.Flush() }

// ----- Reader (client -> server) for Arrays of Bulk Strings (Commands) -----

var (
	ErrProtocol = errors.New("resp: protocol error")
)

// ReadArrayBulk reads a RESP2 Array of Bulk Strings (e.g. *3\r\n$3\r\nSET\r\n$1\r\na\r\n$1\r\nb\r\n)
// Returns slice of []byte with command name and args upper layers can use.
// This is the only form produced by standard Redis clients for commands.
func ReadArrayBulk(r *bufio.Reader) ([][]byte, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if prefix != '*' {
		return nil, fmt.Errorf("%w: expected array '*', got %q", ErrProtocol, prefix)
	}
	n, err := readIntegerLine(r)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		// Null array not expected for commands; treat as protocol error.
		return nil, ErrProtocol
	}
	out := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		// Expect Bulk String
		pfx, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if pfx != '$' {
			return nil, fmt.Errorf("%w: expected bulk '$', got %q", ErrProtocol, pfx)
		}
		size, err := readIntegerLine(r)
		if err != nil {
			return nil, err
		}
		if size == -1 {
			// Null bulk string -> represent as nil to caller
			out = append(out, nil)
			continue
		}
		if size < 0 {
			return nil, ErrProtocol
		}
		buf := make([]byte, size)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		// trailing CRLF
		if err := expectCRLF(r); err != nil {
			return nil, err
		}
		out = append(out, buf)
	}
	return out, nil
}

func readIntegerLine(r *bufio.Reader) (int, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return 0, err
	}
	// line includes trailing '\n'; previous byte should be '\r'
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return 0, fmt.Errorf("%w: expected '$', got %q", ErrProtocol, line)
	}
	// strip \r\n
	line = line[:len(line)-2]
	n, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("%w: expected integer, got %q", ErrProtocol, line)
	}
	return n, nil
}

func expectCRLF(r *bufio.Reader) error {
	b1, err := r.ReadByte()
	if err != nil {
		return err
	}
	b2, err := r.ReadByte()
	if err != nil {
		return err
	}
	if b1 != '\r' || b2 != '\n' {
		return fmt.Errorf("%w: expected CRLF, got %q%q", ErrProtocol, b1, b2)
	}
	return nil
}
