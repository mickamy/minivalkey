package resp

import (
	"fmt"
	"strings"
)

type Arg []byte

type Args []Arg

func (as Args) Cmd() Command {
	if len(as) == 0 {
		return ""
	}
	return Command(strings.ToUpper(string(as[0])))
}

type Command string

func (c Command) String() string {
	return string(c)
}

func WrongNumberOfArgsError(cmd Command) string {
	return fmt.Sprintf("ERR wrong number of arguments for '%s' command", strings.ToLower(cmd.String()))
}

func UnknownCommandError(cmd Command, args Args) string {
	s := fmt.Sprintf("ERR unknown command `%s`, with args beginning with: ", cmd)
	if len(args) > 20 {
		args = args[:20]
	}
	for _, a := range args {
		s += fmt.Sprintf("`%s`, ", string(a))
	}
	return s
}
