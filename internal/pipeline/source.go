package pipeline

import (
	"context"
	"log/slog"
)

func Source(ctx context.Context, inputs []RawURL, logger *slog.Logger) <-chan RawURL {
	return source(ctx, inputs, logger, 0)
}

func source(ctx context.Context, inputs []RawURL, logger *slog.Logger, bufferSize int) <-chan RawURL {
	output := make(chan RawURL, bufferSize)
	go func() {
		defer close(output)
		for _, url := range inputs {
			select {
			case <-ctx.Done():
				if logger != nil {
					logger.Info("source canceled")
				}
				return
			case output <- url:
				if logger != nil {
					logger.Info("source emitted")
				}
			}
		}
	}()
	return output
}
