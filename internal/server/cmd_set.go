package server

import (
	"time"

	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdSet(cmd resp.Command, args resp.Args, w *resp.Writer) error {
	if err := s.validateCommand(cmd, args, validateArgCountAtLeast(3)); err != nil {
		return w.WriteErrorAndFlush(err)
	}

	key := string(args[1])
	val := string(args[2])

	// MVP: ignore EX/PX/NX/XX/KEEPTTL options for now.
	s.store.SetString(key, val, time.Time{})
	if err := w.WriteString("OK"); err != nil {
		return err
	}

	return nil
}
