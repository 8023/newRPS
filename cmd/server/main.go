package main

import (
	"fmt"
	"os"

	"github.com/doumiao/newRPS/internal/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start server: %v\n", err)
		os.Exit(1)
	}
	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
