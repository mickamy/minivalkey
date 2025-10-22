package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdDel(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountAtLeast(2)); err != nil {
		return w.WriteErrorAndFlush(err)
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
