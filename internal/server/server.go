package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/store"
)

// Server wraps a raw TCP listener and processes RESP2 commands.
// One goroutine per accepted connection; each has its own bufio Reader/Writer.
type Server struct {
	listener net.Listener
	doneCh   chan struct{}

	store *store.Store
	now   func() time.Time
}

// New wires a Store and clock fn to a net.Listener.
func New(ln net.Listener, st *store.Store, now func() time.Time) *Server {
	return &Server{
		listener: ln,
		doneCh:   make(chan struct{}),
		store:    st,
		now:      now,
	}
}

// Serve accepts connections and spawns handlers until the listener is closed.
func (s *Server) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Listener closed: exit loop.
			break
		}
		go s.handleConn(conn)
	}
	close(s.doneCh)
}

// Done closes when Serve() exits (useful for coordinating shutdown).
func (s *Server) Done() <-chan struct{} { return s.doneCh }

// Close stops accepting new connections and closes the listener.
func (s *Server) Close() error {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	return nil
}

func (s *Server) handleConn(c net.Conn) {
	defer func(c net.Conn) {
		_ = c.Close()
	}(c)

	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)

	for {
		args, err := resp.ReadArrayBulk(r)
		if err != nil {
			// Client closed or protocol error; end connection.
			return
		}
		if len(args) == 0 || args[0] == nil {
			_ = resp.WriteError(w, "ERR empty command")
			continue
		}
		name := strings.ToUpper(string(args[0]))

		switch name {
		case "PING":
			switch len(args) {
			case 1:
				_ = resp.WriteSimpleString(w, "PONG")
			case 2:
				_ = resp.WriteBulk(w, args[1])
			default:
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'PING'")
			}

		case "HELLO":
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
			if err := resp.WriteArrayHeader(w, 14); err != nil {
				return
			}
			_ = resp.WriteBulkElem(w, []byte("server"))
			_ = resp.WriteBulkElem(w, []byte("valkey"))
			_ = resp.WriteBulkElem(w, []byte("version"))
			_ = resp.WriteBulkElem(w, []byte("0.0.0"))
			_ = resp.WriteBulkElem(w, []byte("proto"))
			_ = resp.WriteIntElem(w, 2) // we speak RESP2
			_ = resp.WriteBulkElem(w, []byte("id"))
			_ = resp.WriteIntElem(w, 1) // arbitrary positive id
			_ = resp.WriteBulkElem(w, []byte("mode"))
			_ = resp.WriteBulkElem(w, []byte("standalone"))
			_ = resp.WriteBulkElem(w, []byte("role"))
			_ = resp.WriteBulkElem(w, []byte("master"))
			_ = resp.WriteBulkElem(w, []byte("modules"))
			_ = resp.WriteEmptyArray(w) // writes "*0\r\n" and Flushes
			_ = resp.Flush(w)

		case "SET":
			if len(args) < 3 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'SET'")
				continue
			}
			key := string(args[1])
			val := string(args[2])

			// MVP: ignore EX/PX/NX/XX/KEEPTTL options for now.
			s.store.SetString(key, val, time.Time{})
			_ = resp.WriteSimpleString(w, "OK")

		case "GET":
			if len(args) != 2 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'GET'")
				continue
			}
			key := string(args[1])
			if v, ok := s.store.GetString(s.now(), key); ok {
				_ = resp.WriteBulk(w, []byte(v))
			} else {
				_ = resp.WriteNull(w)
			}

		case "DEL":
			if len(args) < 2 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'DEL'")
				continue
			}
			keys := make([]string, 0, len(args)-1)
			for _, a := range args[1:] {
				keys = append(keys, string(a))
			}
			n := s.store.Del(keys...)
			_ = resp.WriteInt(w, int64(n))

		case "EXPIRE":
			if len(args) != 3 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'EXPIRE'")
				continue
			}
			key := string(args[1])
			sec, ok := parseInt(args[2])
			if !ok {
				_ = resp.WriteError(w, "ERR value is not an integer or out of range")
				continue
			}
			if s.store.Expire(s.now(), key, sec) {
				_ = resp.WriteInt(w, 1)
			} else {
				_ = resp.WriteInt(w, 0)
			}

		case "TTL":
			if len(args) != 2 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'TTL'")
				continue
			}
			key := string(args[1])
			ttl := s.store.TTL(s.now(), key)
			_ = resp.WriteInt(w, ttl)

		default:
			_ = resp.WriteError(w, fmt.Sprintf("ERR unknown command: '%s'", name))
		}
	}
}

func parseInt(b []byte) (int64, bool) {
	var n int64
	var neg bool
	if len(b) == 0 {
		return 0, false
	}
	for i, c := range b {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}
