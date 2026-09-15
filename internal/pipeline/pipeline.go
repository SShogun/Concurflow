package pipeline

import (
	"context"
	"fmt"
	"log/slog"
)

func Run(ctx context.Context, logger *slog.Logger, input []RawURL) ([]NormalizedURL, error) {
	return RunBuffered(ctx, logger, input, 0)
}

func RunBuffered(ctx context.Context, logger *slog.Logger, input []RawURL, bufferSize int) ([]NormalizedURL, error) {
	if bufferSize < 0 {
		return nil, fmt.Errorf("pipeline buffer size cannot be negative")
	}

	sourced := source(ctx, input, logger, bufferSize)
	transformed := transform(ctx, sourced, logger, bufferSize)
	result, err := Sink(ctx, transformed, logger)
	if err != nil {
		if logger != nil {
			logger.Error("failed to sink URLs", "error", err)
		}
		return nil, err
	}
	if logger != nil {
		logger.Info("pipeline completed", "result_count", len(result))
	}
	return result, nil
}
