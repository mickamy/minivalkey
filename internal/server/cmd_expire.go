package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdExpire(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountExact(3)); err != nil {
		return w.WriteErrorAndFlush(err)
	}
	key := string(args[1])
	sec, ok := resp.ParseInt(args[2])
	if !ok {
		return w.WriteErrorAndFlush(ErrValueNotInteger)
	}
	if s.store.Expire(s.Now(), key, sec) {
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
