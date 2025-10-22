package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/store"
)

// Server wraps a raw TCP listener and processes RESP2 commands.
// One goroutine per accepted connection; each has its own bufio Reader/Writer.
type Server struct {
	listener net.Listener
	doneCh   chan struct{}

	store *store.Store
	clock *clock.Clock
}

// New wires a Store to a net.Listener and seeds the simulated clock.
func New(ln net.Listener, st *store.Store, clk *clock.Clock) (*Server, error) {
	if ln == nil {
		return nil, errors.New("listener is nil")
	}
	if st == nil {
		return nil, errors.New("store is nil")
	}
	if clk == nil {
		return nil, errors.New("clock is nil")
	}
	return &Server{
		listener: ln,
		doneCh:   make(chan struct{}),
		store:    st,
		clock:    clk,
	}, nil
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
				_ = resp.WriteString(w, "PONG")
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

		case "INFO":
			// RESP2: INFO [section]
			// We support sections: "server", "memory", "keyspace", plus "all"/"default".
			// Unknown sections -> error (to match Redis/Valkey behavior).
			section := "default"
			if len(args) == 2 {
				section = strings.ToLower(string(args[1]))
			}
			// Build content based on requested section.
			now := s.Now()
			txt, ok := buildInfo(section, now, s.store, s.uptimeSeconds(now))
			if !ok {
				_ = resp.WriteError(w, "ERR unknown section")
				break
			}
			_ = resp.WriteBulk(w, []byte(txt))

		case "SET":
			if len(args) < 3 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'SET'")
				continue
			}
			key := string(args[1])
			val := string(args[2])

			// MVP: ignore EX/PX/NX/XX/KEEPTTL options for now.
			s.store.SetString(key, val, time.Time{})
			_ = resp.WriteString(w, "OK")

		case "GET":
			if len(args) != 2 {
				_ = resp.WriteError(w, "ERR wrong number of arguments for 'GET'")
				continue
			}
			key := string(args[1])
			if v, ok := s.store.GetString(s.Now(), key); ok {
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
			if s.store.Expire(s.Now(), key, sec) {
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
			ttl := s.store.TTL(s.Now(), key)
			_ = resp.WriteInt(w, ttl)

		default:
			_ = resp.WriteError(w, fmt.Sprintf("ERR unknown command: '%s'", name))
		}
	}
}

// uptimeSeconds returns server uptime in seconds based on the simulated clock.
func (s *Server) uptimeSeconds(now time.Time) int64 {
	return int64(now.Sub(s.clock.Base()).Seconds())
}

// Now returns the current simulated time for the server.
func (s *Server) Now() time.Time {
	return s.clock.Now()
}

// AdvanceClock moves the simulated clock forward and returns the updated time.
func (s *Server) AdvanceClock(d time.Duration) time.Time {
	return s.clock.Advance(d)
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

// buildInfo builds an INFO string for a given section.
// Returns (text, true) if section is supported; ("", false) otherwise.
func buildInfo(section string, now time.Time, st *store.Store, uptimeSec int64) (string, bool) {
	switch section {
	case "all", "default":
		var b strings.Builder
		b.WriteString(infoServer(now, uptimeSec))
		b.WriteString(infoMemory(now, st))
		b.WriteString(infoKeyspace(now, st))
		return b.String(), true
	case "server":
		return infoServer(now, uptimeSec), true
	case "memory":
		return infoMemory(now, st), true
	case "keyspace":
		return infoKeyspace(now, st), true
	case "replication":
		return infoReplication(), true
	default:
		fmt.Println("unknown info section:", section)
		return "", false
	}
}

func infoServer(now time.Time, uptimeSec int64) string {
	var b strings.Builder
	b.WriteString("# Server\r\n")
	// server: string identifier (we advertise "valkey" for compatibility)
	b.WriteString("server:valkey\r\n")
	// version: library version; keep "0.0.0" for now, or wire to a const
	b.WriteString("version:0.0.0\r\n")
	// proto: we speak RESP2
	b.WriteString("proto:2\r\n")
	// process_id: arbitrary positive id (we don't fork, so constant is fine)
	b.WriteString("process_id:1\r\n")
	// uptime_in_seconds: based on simulated clock
	b.WriteString("uptime_in_seconds:")
	b.WriteString(strconv.FormatInt(uptimeSec, 10))
	b.WriteString("\r\n")
	// mode/role: single node master-like
	b.WriteString("mode:standalone\r\n")
	b.WriteString("role:master\r\n")
	// time_now: unix seconds (simulated clock)
	b.WriteString("time_now:")
	b.WriteString(strconv.FormatInt(now.Unix(), 10))
	b.WriteString("\r\n\r\n")
	return b.String()
}

func infoMemory(now time.Time, st *store.Store) string {
	keys, expires, _ := st.Stats(now)
	var b strings.Builder
	b.WriteString("# Memory\r\n")
	// used_memory: we don't track bytes; expose number of keys as a hint
	b.WriteString("used_memory_keys:")
	b.WriteString(strconv.Itoa(keys))
	b.WriteString("\r\n")
	// expires: number of keys with TTL
	b.WriteString("expires:")
	b.WriteString(strconv.Itoa(expires))
	b.WriteString("\r\n\r\n")
	return b.String()
}

func infoKeyspace(now time.Time, st *store.Store) string {
	keys, expires, avgTTLms := st.Stats(now)
	var b strings.Builder
	b.WriteString("# Keyspace\r\n")
	// Only emit db0 if there are any keys (mimic Redis behavior)
	if keys > 0 {
		// format: db0:keys=<int>,expires=<int>,avg_ttl=<milliseconds>
		b.WriteString("db0:")
		b.WriteString("keys=")
		b.WriteString(strconv.Itoa(keys))
		b.WriteString(",expires=")
		b.WriteString(strconv.Itoa(expires))
		b.WriteString(",avg_ttl=")
		b.WriteString(strconv.FormatInt(avgTTLms, 10))
		b.WriteString("\r\n")
	}
	b.WriteString("\r\n")
	return b.String()
}

func infoReplication() string {
	// Minimal, master-only, no backlog. Enough for clients probing replication.
	var b strings.Builder
	b.WriteString("# Replication\r\n")
	b.WriteString("role:master\r\n")
	b.WriteString("connected_slaves:0\r\n")
	// 40 hex chars, dummy but valid-shaped replid values
	b.WriteString("master_replid:0000000000000000000000000000000000000000\r\n")
	b.WriteString("master_replid2:0000000000000000000000000000000000000000\r\n")
	b.WriteString("master_repl_offset:0\r\n")
	b.WriteString("second_repl_offset:-1\r\n")
	b.WriteString("repl_backlog_active:0\r\n\r\n")
	return b.String()
}
