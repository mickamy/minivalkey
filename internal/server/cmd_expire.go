package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdExpire(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
	if len(args) != 3 {
		if err := w.WriteError("ERR wrong number of arguments for 'EXPIRE'"); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
	}
	key := string(args[1])
	sec, ok := parseInt(args[2])
	if !ok {
		if err := w.WriteError("ERR value is not an integer or out of range"); err != nil {
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
