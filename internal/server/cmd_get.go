package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdGet(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
	if len(args) != 2 {
		if err := w.WriteError("ERR wrong number of arguments for 'GET'"); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
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
