package server

import (
	"github.com/mickamy/minivalkey/internal/resp"
	"github.com/mickamy/minivalkey/internal/session"
)

// request represents a client request to the server.
type request struct {
	session *session.Session
	cmd     resp.Command
	args    resp.Args
}

func newRequest(sess *session.Session, cmd resp.Command, args resp.Args) *request {
	return &request{
		session: sess,
		cmd:     cmd,
		args:    args,
	}
}
