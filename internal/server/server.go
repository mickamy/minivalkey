package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/logger"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/store"
)

type handleFunc func(cmd resp.Command, args resp.Args, w *resp.Writer) error

// Server wraps a raw TCP listener and processes RESP2 commands.
// One goroutine per accepted connection; each has its own bufio Reader/Writer.
type Server struct {
	listener net.Listener
	doneCh   chan struct{}

	store *store.Store
	clock *clock.Clock

	cmds map[string]handleFunc
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
	s := &Server{
		listener: ln,
		doneCh:   make(chan struct{}),
		store:    st,
		clock:    clk,
		cmds:     make(map[string]handleFunc),
	}

	handlers := map[string]handleFunc{
		"DEL":    s.cmdDel,
		"EXISTS": s.cmdExists,
		"EXPIRE": s.cmdExpire,
		"GET":    s.cmdGet,
		"HELLO":  s.cmdHello,
		"INFO":   s.cmdInfo,
		"PING":   s.cmdPing,
		"SET":    s.cmdSet,
		"TTL":    s.cmdTTL,
	}
	for cmd, handler := range handlers {
		if err := s.register(cmd, handler); err != nil {
			return nil, fmt.Errorf("failed to register command %s: %w", cmd, err)
		}
	}

	return s, nil
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
	return s.listener.Close()
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
			if err := w.WriteErrorAndFlush(ErrEmptyCommand); err != nil {
				logger.Error("failed to write and flush error", "err", err)
				return
			}
			continue
		}

		cmd := args.Cmd()
		handle, ok := s.cmds[cmd.String()]
		if !ok {
			logger.Warn("unknown command", "cmd", cmd)

			if err := w.WriteErrorAndFlush(errors.New(resp.UnknownCommandError(cmd, args))); err != nil {
				logger.Error("failed to write and flush error", "err", err)
				return
			}
			continue
		}

		if err := handle(cmd, args, w); err != nil {
			logger.Error("command handler error", "cmd", cmd.String(), "err", err)
			return
		}
		if err := w.Flush(); err != nil {
			logger.Error("failed to flush writer", "err", err)
			return
		}
	}
}

func (s *Server) register(name string, handle handleFunc) error {
	if _, exists := s.cmds[name]; exists {
		return fmt.Errorf("command %s already exists", name)
	}
	s.cmds[name] = handle
	return nil
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
