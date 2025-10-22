package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdPing(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
	switch len(args) {
	case 1:
		if err := w.WriteString("PONG"); err != nil {
			return err
		}
	case 2:
		if err := w.WriteBulk(args[1]); err != nil {
			return err
		}
	default:
		if err := w.WriteError(cmd.WrongNumberOfArgsError()); err != nil {
			return err
		}
	}

	return nil
}
