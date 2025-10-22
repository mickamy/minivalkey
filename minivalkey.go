package minivalkey

import (
	"net"
	"time"

	"github.com/mickamy/minivalkey/internal/clock"
	"github.com/mickamy/minivalkey/internal/server"
	"github.com/mickamy/minivalkey/internal/store"
)

// MiniValkey represents an in-memory Valkey-compatible server instance.
// It provides APIs for starting, stopping, and manipulating simulated time.
type MiniValkey struct {
	addr  string
	srv   *server.Server
	store *store.Store
}

// Run starts a new in-memory Valkey server listening on an ephemeral port.
func Run() (*MiniValkey, error) {
	st := store.New()
	s := &MiniValkey{
		store: st,
	}
	clk := clock.New(time.Now())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s.addr = ln.Addr().String()

	// Start TCP server
	s.srv, err = server.New(ln, st, clk)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}

	go s.srv.Serve()

	// Background clean-up for expired keys
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.srv.Done():
				return
			case <-ticker.C:
				st.CleanUpExpired(s.srv.Now())
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
	// Advance simulated clock inside the server, then run cleanup outside the lock.
	now := s.srv.AdvanceClock(d)
	s.store.CleanUpExpired(now)
}
