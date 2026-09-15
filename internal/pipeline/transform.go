package pipeline

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
)

func Transform(ctx context.Context, in <-chan RawURL, logger *slog.Logger) <-chan NormalizedURL {
	return transform(ctx, in, logger, 0)
}

func transform(ctx context.Context, in <-chan RawURL, logger *slog.Logger, bufferSize int) <-chan NormalizedURL {
	output := make(chan NormalizedURL, bufferSize)
	go func() {
		defer close(output)

		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-in:
				if !ok {
					return
				}

				normalized := normalize(raw, logger)
				select {
				case <-ctx.Done():
					return
				case output <- normalized:
				}
			}
		}
	}()
	return output
}

func normalize(raw RawURL, logger *slog.Logger) NormalizedURL {
	trimmed := strings.TrimSpace(raw.URL)
	result := NormalizedURL{ID: raw.ID, URL: trimmed}

	if trimmed == "" {
		result.Reason = []Reason{empty}
		if logger != nil {
			logger.Info("empty URL", "id", raw.ID)
		}
		return result
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		result.Reason = []Reason{malformedURL}
		if logger != nil {
			logger.Info("malformed URL", "id", raw.ID, "error", err)
		}
		return result
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme == "" {
		result.Reason = []Reason{missingScheme}
		if logger != nil {
			logger.Info("missing scheme", "id", raw.ID, "url", trimmed)
		}
		return result
	}
	if scheme != "http" && scheme != "https" {
		result.Reason = []Reason{unsupportedScheme}
		if logger != nil {
			logger.Info("unsupported scheme", "id", raw.ID, "scheme", u.Scheme)
		}
		return result
	}
	if u.Host == "" {
		result.Reason = []Reason{missingHost}
		if logger != nil {
			logger.Info("missing host", "id", raw.ID, "url", trimmed)
		}
		return result
	}

	result.Valid = true
	result.Reason = []Reason{fair}
	if logger != nil {
		logger.Info("transformed URL", "id", raw.ID, "url", trimmed)
	}
	return result
}
