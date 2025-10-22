package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdGet(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountExact(2)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	key := string(args[1])
	if v, ok := s.store.GetString(s.Now(), key); ok {
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
