package pipeline

import (
	"context"
	"log/slog"
)

func Sink(ctx context.Context, in <-chan NormalizedURL, logger *slog.Logger) ([]NormalizedURL, error) {
	var result []NormalizedURL
	for {
		select {
		case <-ctx.Done():
			logger.Info("sink canceled")
			return nil, ctx.Err()
		case url, ok := <-in:
			if !ok {
				logger.Info("sink completed", "count", len(result))
				return result, nil
			}
			result = append(result, url)
		}
	}
}
