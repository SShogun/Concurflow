package pipeline

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
)

func Transform(ctx context.Context, in <-chan RawURL, logger *slog.Logger) <-chan NormalizedURL {
	output := make(chan NormalizedURL)
	go func() {
		defer close(output)
		for {
			var text RawURL
			var ok bool

			select {
			case <-ctx.Done():
				return
			case text, ok = <-in:
				if !ok {
					return
				}
			}

			trimmed := strings.TrimSpace(text.URL)
			if trimmed == "" {
				logger.Info("empty URL", "id", text.ID)
				select {
				case <-ctx.Done():
					return
				case output <- NormalizedURL{
					ID:     text.ID,
					URL:    trimmed,
					Valid:  false,
					Reason: []Reason{empty},
				}:
				}
				continue
			}

			u, err := url.Parse(trimmed)
			if err != nil {
				logger.Info("invalid URL", "id", text.ID, "error", err)
				select {
				case <-ctx.Done():
					return
				case output <- NormalizedURL{
					ID:     text.ID,
					URL:    trimmed,
					Valid:  false,
					Reason: []Reason{missing_scheme},
				}:
				}
				continue
			}

			if u.Scheme == "" {
				logger.Info("missing scheme", "id", text.ID, "url", trimmed)
				select {
				case <-ctx.Done():
					return
				case output <- NormalizedURL{
					ID:     text.ID,
					URL:    trimmed,
					Valid:  false,
					Reason: []Reason{missing_scheme},
				}:
				}
				continue
			}

			if u.Scheme != "http" && u.Scheme != "https" {
				logger.Info("unsupported scheme", "id", text.ID, "scheme", u.Scheme)
				select {
				case <-ctx.Done():
					return
				case output <- NormalizedURL{
					ID:     text.ID,
					URL:    trimmed,
					Valid:  false,
					Reason: []Reason{unsupported_scheme},
				}:
				}
				continue
			}

			select {
			case <-ctx.Done():
				return
			case output <- NormalizedURL{
				ID:     text.ID,
				URL:    trimmed,
				Valid:  true,
				Reason: []Reason{fair},
			}:
				logger.Info("transformed URL", "id", text.ID, "url", trimmed)
			}
		}
	}()
	return output
}
