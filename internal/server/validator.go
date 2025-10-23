package server

import (
	"errors"

	"github.com/mickamy/minivalkey/internal/resp"
)

type cmdValidator func(cmd resp.Command, args resp.Args) error

func validateArgCountExact(expected int) cmdValidator {
	return func(cmd resp.Command, args resp.Args) error {
		if len(args) != expected {
			return errors.New(resp.WrongNumberOfArgsError(cmd))
		}
		return nil
	}
}

func validateArgCountAtLeast(min int) cmdValidator {
	return func(cmd resp.Command, args resp.Args) error {
		if len(args) < min {
			return errors.New(resp.WrongNumberOfArgsError(cmd))
		}
		return nil
	}
}

func validateArgCountAtMost(max int) cmdValidator {
	return func(cmd resp.Command, args resp.Args) error {
		if len(args) > max {
			return errors.New(resp.WrongNumberOfArgsError(cmd))
		}
		return nil
	}
}

func validateCommand(cmd resp.Command, args resp.Args, validators ...cmdValidator) error {
	for _, v := range validators {
		if err := v(cmd, args); err != nil {
			return err
		}
	}
	return nil
}
