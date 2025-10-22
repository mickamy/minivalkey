package server

import (
	"fmt"

	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdHello(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	// Minimal HELLO handler:
	// - Accepts "HELLO", "HELLO 2", and "HELLO 3".
	// - Always negotiates RESP2 (proto=2) and returns a map as alternating key/value array.
	// - Ignores other HELLO options for now (AUTH, SETNAME, etc.).
	wantProto := 0
	if len(args) >= 2 {
		// If arg[1] is a number (e.g., "2" or "3"), accept it but we'll still serve RESP2.
		if n, ok := parseInt(args[1]); ok {
			wantProto = int(n) // kept only for debugging; we don't switch to RESP3
			_ = wantProto
		}
		// else: could be keywords like "AUTH", "SETNAME" — ignore for MVP
	}
	// Build RESP2-style map as alternating key/value array:
	// ["server","valkey","version","0.0.0","proto",2,"id",1,"mode","standalone","role","master","modules",[]]
	if err := w.WriteArrayHeader(14); err != nil {
		return err
	}

	type emptyArray struct{}
	fields := []struct {
		key   string
		value any
	}{
		{"server", "valkey"},
		{"version", "0.0.0"},
		{"proto", int64(2)},
		{"id", int64(1)},
		{"mode", "standalone"},
		{"role", "master"},
		{"modules", emptyArray{}},
	}

	for _, field := range fields {
		if err := w.WriteBulkElem([]byte(field.key)); err != nil {
			return err
		}
		switch v := field.value.(type) {
		case string:
			if err := w.WriteBulkElem([]byte(v)); err != nil {
				return err
			}
		case int64:
			if err := w.WriteIntElem(v); err != nil {
				return err
			}
		case emptyArray:
			if err := w.WriteEmptyArray(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported HELLO field type %T", v)
		}
	}

	return nil
}
