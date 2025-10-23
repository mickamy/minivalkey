package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdDel(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountAtLeast(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	keys := make([]string, len(r.args)-1)
	for i, a := range r.args[1:] {
		keys[i] = string(a)
	}

	n := s.db(r.session).Del(keys...)
	if err := w.WriteInt(int64(n)); err != nil {
		return err
	}

	return nil
}
