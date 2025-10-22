package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdTTL(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
	if len(args) != 2 {
		if err := w.WriteError("ERR wrong number of arguments for 'TTL'"); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
	}
	key := string(args[1])
	ttl := s.store.TTL(s.Now(), key)
	if err := w.WriteInt(ttl); err != nil {
		return err
	}

	return nil
}
