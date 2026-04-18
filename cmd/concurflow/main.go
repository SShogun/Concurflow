package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"Concurflow/internal/app"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := app.DefaultConfig()
	a := app.New(cfg)

	if err := a.Run(ctx); err != nil {
		os.Exit(1)
	}
}
