package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdPing(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountAtMost(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}

	switch len(r.args) {
	case 1:
		if err := w.WriteString("PONG"); err != nil {
			return err
		}
	case 2:
		if err := w.WriteBulk(r.args[1]); err != nil {
			return err
		}
	}

	return nil
}
