package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdHello(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
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
	ops := []func() error{
		func() error { return w.WriteArrayHeader(14) },
		func() error { return w.WriteBulkElem([]byte("server")) },
		func() error { return w.WriteBulkElem([]byte("valkey")) },
		func() error { return w.WriteBulkElem([]byte("version")) },
		func() error { return w.WriteBulkElem([]byte("0.0.0")) },
		func() error { return w.WriteBulkElem([]byte("proto")) },
		func() error { return w.WriteIntElem(2) },
		func() error { return w.WriteBulkElem([]byte("id")) },
		func() error { return w.WriteIntElem(1) },
		func() error { return w.WriteBulkElem([]byte("mode")) },
		func() error { return w.WriteBulkElem([]byte("standalone")) },
		func() error { return w.WriteBulkElem([]byte("role")) },
		func() error { return w.WriteBulkElem([]byte("master")) },
		func() error { return w.WriteBulkElem([]byte("modules")) },
		func() error { return w.WriteEmptyArray() },
	}
	for _, op := range ops {
		if err := op(); err != nil {
			return err
		}
	}

	return nil
}
