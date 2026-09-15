package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/SShogun/Concurflow/internal/app"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := app.DefaultConfig()
	a := app.New(cfg)
	return a.Run(ctx)
}
