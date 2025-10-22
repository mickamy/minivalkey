package minivalkey

import (
	"net"
	"sync"
	"time"

	"github.com/mickamy/minivalkey/internal/server"
	"github.com/mickamy/minivalkey/internal/store"
)

// MiniValkey represents an in-memory Valkey-compatible server instance.
// It provides APIs for starting, stopping, and manipulating simulated time.
type MiniValkey struct {
	addr      string
	srv       *server.Server
	store     *store.Store
	clockMu   sync.RWMutex
	clockBase time.Time
	offset    time.Duration
}

// Run starts a new in-memory Valkey server listening on an ephemeral port.
func Run() (*MiniValkey, error) {
	st := store.New()
	s := &MiniValkey{
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
func (s *MiniValkey) Addr() string { return s.addr }

// Host returns the host part of the server address.
func (s *MiniValkey) Host() string {
	host, _, _ := net.SplitHostPort(s.addr)
	return host
}

// Port returns the port part of the server address.
func (s *MiniValkey) Port() string {
	_, port, _ := net.SplitHostPort(s.addr)
	return port
}

// Close stops the server and releases resources.
func (s *MiniValkey) Close() error {
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

// FastForward advances the internal clock by the specified duration.
// Useful for testing key expiration.
func (s *MiniValkey) FastForward(d time.Duration) {
	// 1) lock, update offset, compute "now" locally, then unlock
	s.clockMu.Lock()
	s.offset += d
	now := s.clockBase.Add(s.offset)
	s.clockMu.Unlock()

	// 2) run expiration cleanup *outside* the clock lock
	s.store.CleanUpExpired(now)
}

// now returns the current simulated time.
func (s *MiniValkey) now() time.Time {
	s.clockMu.RLock()
	defer s.clockMu.RUnlock()
	return s.clockBase.Add(s.offset)
}
