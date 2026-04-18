package pipeline

import (
	"context"
	"log/slog"
)

func Run(ctx context.Context, logger *slog.Logger, input []RawURL) ([]NormalizedURL, error) {
	sourced := Source(ctx, input, logger)
	transformed := Transform(ctx, sourced, logger)
	result, err := Sink(ctx, transformed, logger)
	if err != nil {
		logger.Error("failed to sink URLs", "error", err)
		return nil, err
	}
	logger.Info("pipeline completed", "result", result)
	return result, nil
}
