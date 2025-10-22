package server

import (
	"errors"

	"github.com/mickamy/minivalkey/internal/resp"
)

var (
	ErrExpireValueNotInteger = errors.New("ERR value is not an integer or out of range")
)

func (s *Server) cmdExpire(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountExact(3)); err != nil {
		if err := w.WriteError(err); err != nil {
			return err
		}
		return w.Flush()
	}
	key := string(args[1])
	sec, ok := parseInt(args[2])
	if !ok {
		if err := w.WriteError(ErrExpireValueNotInteger); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
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
