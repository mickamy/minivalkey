package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdPing(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountAtMost(2)); err != nil {
		if err := w.WriteError(err); err != nil {
			return err
		}
		return w.Flush()
	}

	switch len(args) {
	case 1:
		if err := w.WriteString("PONG"); err != nil {
			return err
		}
	case 2:
		if err := w.WriteBulk(args[1]); err != nil {
			return err
		}
	}

	return nil
}
