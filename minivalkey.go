package minivalkey

import (
	"net"
	"sync"
	"time"

	"github.com/mickamy/minivalkey/internal/server"
	"github.com/mickamy/minivalkey/internal/store"
)

// Server represents an in-memory Valkey-compatible server instance.
// It provides APIs for starting, stopping, and manipulating simulated time.
type Server struct {
	addr      string
	srv       *server.Server
	store     *store.Store
	clockMu   sync.RWMutex
	clockBase time.Time
	offset    time.Duration
}

// Run starts a new in-memory Valkey server listening on an ephemeral port.
func Run() (*Server, error) {
	st := store.New()
	s := &Server{
		store:     st,
		clockBase: time.Now(),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s.addr = ln.Addr().String()

	// Start TCP server
	s.srv = server.New(ln, st, s.now)
	go s.srv.Serve()

	// Background cleanup for expired keys
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.srv.Done():
				return
			case <-ticker.C:
				st.CleanUpExpired(s.now())
			}
		}
	}()

	return s, nil
}

// Addr returns the TCP address of the running server.
func (s *Server) Addr() string { return s.addr }

// Close stops the server and releases resources.
func (s *Server) Close() error {
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

// FastForward advances the internal clock by the specified duration.
// Useful for testing key expiration.
func (s *Server) FastForward(d time.Duration) {
	s.clockMu.Lock()
	defer s.clockMu.Unlock()
	s.offset += d
	s.store.CleanUpExpired(s.now())
}

// now returns the current simulated time.
func (s *Server) now() time.Time {
	s.clockMu.RLock()
	defer s.clockMu.RUnlock()
	return s.clockBase.Add(s.offset)
}
