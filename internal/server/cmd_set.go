package server

import (
	"time"

	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdSet(cmd resp.Cmd, args resp.Args, w *resp.Writer) error {
	if len(args) < 3 {
		if err := w.WriteError("ERR wrong number of arguments for 'SET'"); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
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
