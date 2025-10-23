package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdGet(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountExact(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	key := string(r.args[1])
	if v, ok := s.db(r.session).GetString(s.Now(), key); ok {
		if err := w.WriteBulk([]byte(v)); err != nil {
			return err
		}
	} else {
		if err := w.WriteNull(); err != nil {
			return err
		}
	}

	return nil
}
