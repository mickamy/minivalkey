package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdTTL(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountExact(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	key := string(r.args[1])
	ttl := s.db(r.session).TTL(s.Now(), key)
	if err := w.WriteInt(ttl); err != nil {
		return err
	}

	return nil
}
