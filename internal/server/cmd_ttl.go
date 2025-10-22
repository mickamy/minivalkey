package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdTTL(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountExact(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	key := string(args[1])
	ttl := s.store.TTL(s.Now(), key)
	if err := w.WriteInt(ttl); err != nil {
		return err
	}

	return nil
}
