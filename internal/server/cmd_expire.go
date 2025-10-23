package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdExpire(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountExact(3)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	key := string(r.args[1])
	sec, ok := resp.ParseInt(r.args[2])
	if !ok {
		return w.WriteErrorAndFlush(ErrValueNotInteger)
	}
	if s.db(r.session).Expire(s.Now(), key, sec) {
		if err := w.WriteInt(1); err != nil {
			return err
		}
	} else {
		if err := w.WriteInt(0); err != nil {
			return err
		}
	}

	return nil
}
