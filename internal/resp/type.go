package resp

import (
	"fmt"
	"strings"
)

type Arg []byte

type Args []Arg

func (as Args) Cmd() Cmd {
	if len(as) == 0 {
		return ""
	}
	return Cmd(strings.ToUpper(string(as[0])))
}

type Cmd string

func (c Cmd) String() string {
	return string(c)
}

func (c Cmd) WrongNumberOfArgsError() string {
	return fmt.Sprintf("ERR wrong number of arguments for '%s' command", strings.ToLower(c.String()))
}

func (c Cmd) UnknownCommandError(args Args) string {
	s := fmt.Sprintf("ERR unknown command `%s`, with args beginning with: ", c.String())
	if len(args) > 20 {
		args = args[:20]
	}
	for _, a := range args {
		s += fmt.Sprintf("`%s`, ", string(a))
	}
	return s
}
