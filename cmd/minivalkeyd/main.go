package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mickamy/minivalkey"
)

func main() {
	s, err := minivalkey.Run()
	if err != nil {
		fmt.Println("failed to start:", err)
		os.Exit(1)
	}
	fmt.Println("minivalkey listening at", s.Addr())
	defer func(s *minivalkey.Server) {
		_ = s.Close()
	}(s)

	// Wait for Ctrl+C
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("shutting down...")
}
