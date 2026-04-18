package pipeline

import (
	"context"
	"log/slog"
)

func Source(ctx context.Context, inputs []RawURL, logger *slog.Logger) <-chan RawURL {
	output := make(chan RawURL)
	go func() {
		defer close(output)
		for _, url := range inputs {
			select {
			case <-ctx.Done():
				logger.Info("source canceled")
				return
			case output <- url:
				logger.Info("source emitted")
			}
		}
	}()
	return output
}
