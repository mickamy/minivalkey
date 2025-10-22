package resp

import (
	"bufio"
	"strconv"
)

// Writer provides RESP2 write helpers over a buffered writer.
type Writer struct {
	w *bufio.Writer
}

// NewWriter wraps the provided bufio.Writer.
func NewWriter(w *bufio.Writer) *Writer {
	return &Writer{w: w}
}

// Flush flushes the underlying writer.
func (w *Writer) Flush() error {
	return w.w.Flush()
}

// WriteString writes a RESP2 simple string ("+...").
func (w *Writer) WriteString(s string) error {
	_, err := w.w.WriteString("+" + s + "\r\n")
	return err
}

// WriteError writes a RESP2 error ("-...").
func (w *Writer) WriteError(msg string) error {
	_, err := w.w.WriteString("-" + msg + "\r\n")
	return err
}

// WriteInt writes a RESP2 integer (":...").
func (w *Writer) WriteInt(n int64) error {
	_, err := w.w.WriteString(":" + strconv.FormatInt(n, 10) + "\r\n")
	return err
}

// WriteBulk writes a RESP2 bulk string ("$...").
func (w *Writer) WriteBulk(b []byte) error {
	if b == nil {
		_, err := w.w.WriteString("$-1\r\n")
		return err
	}
	if _, err := w.w.WriteString("$" + strconv.Itoa(len(b)) + "\r\n"); err != nil {
		return err
	}
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	_, err := w.w.WriteString("\r\n")
	return err
}

// WriteArrayHeader writes a RESP2 array header ("*<n>\r\n").
func (w *Writer) WriteArrayHeader(n int) error {
	_, err := w.w.WriteString("*" + strconv.Itoa(n) + "\r\n")
	return err
}

// WriteBulkElem writes a RESP2 bulk string element without flushing.
func (w *Writer) WriteBulkElem(b []byte) error {
	if b == nil {
		_, err := w.w.WriteString("$-1\r\n")
		return err
	}
	if _, err := w.w.WriteString("$" + strconv.Itoa(len(b)) + "\r\n"); err != nil {
		return err
	}
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	_, err := w.w.WriteString("\r\n")
	return err
}

// WriteIntElem writes a RESP2 integer element without flushing.
func (w *Writer) WriteIntElem(n int64) error {
	_, err := w.w.WriteString(":" + strconv.FormatInt(n, 10) + "\r\n")
	return err
}

// WriteEmptyArray writes a RESP2 empty array ("*0").
func (w *Writer) WriteEmptyArray() error {
	_, err := w.w.WriteString("*0\r\n")
	return err
}

// WriteNull writes a RESP2 null bulk string ("$-1").
func (w *Writer) WriteNull() error {
	_, err := w.w.WriteString("$-1\r\n")
	return err
}
