package server

import (
	"errors"
)

var (
	ErrEmptyCommand    = errors.New("ERR empty command")
	ErrValueNotInteger = errors.New("ERR value is not an integer or out of range")
	ErrUnknownSection  = errors.New("ERR unknown section")
)
