package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdDel(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
	if len(args) < 2 {
		if err := w.WriteError("ERR wrong number of arguments for 'DEL'"); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
	}
	keys := make([]string, 0, len(args)-1)
	for _, a := range args[1:] {
		keys = append(keys, string(a))
	}
	n := s.store.Del(keys...)
	if err := w.WriteInt(int64(n)); err != nil {
		return err
	}

	return nil
}
