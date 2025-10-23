package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdExists(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountAtLeast(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}

	keys := make([]string, len(r.args)-1)
	for i, a := range r.args[1:] {
		keys[i] = string(a)
	}
	count := s.db(r.session).Exists(s.Now(), keys...)
	if err := w.WriteInt(int64(count)); err != nil {
		return err
	}

	return nil
}
