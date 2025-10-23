package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/db"
	"github.com/mickamy/minivalkey/internal/logger"
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/session"
)

type handleFunc func(w *resp.Writer, r *request) error

// Server wraps a raw TCP listener and processes RESP2 commands.
// One goroutine per accepted connection; each has its own bufio Reader/Writer.
type Server struct {
	listener       net.Listener
	doneCh         chan struct{}
	dbMu           sync.RWMutex
	dbMap          map[int]*db.DB
	cleanUpBufPool sync.Pool
	clock          *clock.Clock
	handlers       map[string]handleFunc
}

// New wires a DB to a net.Listener and seeds the simulated clock.
func New(ln net.Listener) (*Server, error) {
	if ln == nil {
		return nil, errors.New("listener is nil")
	}
	s := &Server{
		listener: ln,
		doneCh:   make(chan struct{}),
		dbMap:    make(map[int]*db.DB),
		cleanUpBufPool: sync.Pool{
			New: func() any { return make([]*db.DB, 0, 8) },
		},
		clock:    clock.New(time.Now()),
		handlers: make(map[string]handleFunc),
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
	sess := session.New()

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
		handle, ok := s.handlers[cmd.String()]
		if !ok {
			logger.Warn("unknown command", "cmd", cmd)

			if err := w.WriteErrorAndFlush(errors.New(resp.UnknownCommandError(cmd, args))); err != nil {
				logger.Error("failed to write and flush error", "err", err)
				return
			}
			continue
		}

		req := newRequest(sess, cmd, args)

		if err := handle(w, req); err != nil {
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
	if _, exists := s.handlers[name]; exists {
		return fmt.Errorf("command %s already exists", name)
	}
	s.handlers[name] = handle
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

// FastForward advances the internal clock by the specified duration.
func (s *Server) FastForward(d time.Duration) {
	now := s.clock.Advance(d)
	s.CleanUpExpired(now)
}

// CleanUpExpired removes expired keys based on the current simulated time.
func (s *Server) CleanUpExpired(now time.Time) {
	bufAny := s.cleanUpBufPool.Get()
	var dbs []*db.DB
	if bufAny != nil {
		dbs = bufAny.([]*db.DB)
	}
	s.dbMu.RLock()
	n := len(s.dbMap)
	if cap(dbs) < n {
		dbs = make([]*db.DB, 0, n)
	} else {
		dbs = dbs[:0]
	}
	for _, d := range s.dbMap {
		dbs = append(dbs, d)
	}
	s.dbMu.RUnlock()

	for _, d := range dbs {
		d.CleanUpExpired(now)
	}

	for i := range dbs {
		dbs[i] = nil
	}
	s.cleanUpBufPool.Put(dbs[:0])
}

// db returns the DB instance for the selected database in the session.
func (s *Server) db(sess *session.Session) *db.DB {
	s.dbMu.RLock()
	d, ok := s.dbMap[sess.SelectedDB]
	s.dbMu.RUnlock()
	if ok {
		return d
	}

	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	d, ok = s.dbMap[sess.SelectedDB]
	if ok {
		return d
	}

	d = db.New()
	s.dbMap[sess.SelectedDB] = d
	return d
}
