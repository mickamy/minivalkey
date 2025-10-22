package server

import (
	"bufio"
	"errors"
	"net"
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

	r := resp.NewReader(bufio.NewReader(c))
	w := resp.NewWriter(bufio.NewWriter(c))

	for {
		args, err := r.ReadArrayBulk()
		if err != nil {
			// Client closed or protocol error; end connection.
			return
		}
		if len(args) == 0 || args[0] == nil {
			if err := w.WriteError("ERR empty command"); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
			continue
		}

		cmd := args.Cmd()

		switch cmd {
		case "PING":
			if err := s.cmdPing(cmd, args, w); err != nil {
				return
			}

		case "HELLO":
			if err := s.cmdHello(cmd, args, w); err != nil {
				return
			}

		case "INFO":
			if err := s.cmdInfo(cmd, args, w); err != nil {
				return
			}

		case "SET":
			if err := s.cmdSet(cmd, args, w); err != nil {
				return
			}

		case "GET":
			if err := s.cmdGet(cmd, args, w); err != nil {
				return
			}

		case "DEL":
			if err := s.cmdDel(cmd, args, w); err != nil {
				return
			}

		case "EXPIRE":
			if err := s.cmdExpire(cmd, args, w); err != nil {
				return
			}

		case "TTL":
			if err := s.cmdTTL(cmd, args, w); err != nil {
				return
			}

		default:
			if err := w.WriteError(cmd.UnknownCommandError(args)); err != nil {
				return
			}
		}

		if err := w.Flush(); err != nil {
			return
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
