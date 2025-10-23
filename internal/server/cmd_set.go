package server

import (
	"time"

	"github.com/mickamy/minivalkey/internal/resp"
)

func (s *Server) cmdSet(w *resp.Writer, r *request) error {
	if err := validateCommand(r.cmd, r.args, validateArgCountAtLeast(3)); err != nil {
		return w.WriteErrorAndFlush(err)
	}

	key := string(r.args[1])
	val := string(r.args[2])

	// MVP: ignore EX/PX/NX/XX/KEEPTTL options for now.
	s.db(r.session).SetString(key, val, time.Time{})
	if err := w.WriteString("OK"); err != nil {
		return err
	}

	return nil
}
