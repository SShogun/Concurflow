package downloader

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func fetch(ctx context.Context, client *http.Client, req DownloadRequest, logger *slog.Logger) DownloadResult {
	start := time.Now()
	result := DownloadResult{
		ID:  req.ID,
		URL: req.URL,
	}

	if client == nil {
		client = http.DefaultClient
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		result.Err = err
		result.Duration = time.Since(start)
		if logger != nil {
			logger.Error("failed to create request", "component", "downloader", "url", req.URL, "error", err)
		}
		return result
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		result.Err = err
		result.Duration = time.Since(start)
		if logger != nil {
			switch {
			case errors.Is(err, context.DeadlineExceeded):
				logger.Warn("download timed out", "component", "downloader", "url", req.URL, "error", err)
			case errors.Is(err, context.Canceled):
				logger.Info("download canceled", "component", "downloader", "url", req.URL, "error", err)
			default:
				logger.Error("download failed", "component", "downloader", "url", req.URL, "error", err)
			}
		}
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		result.Err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		if logger != nil {
			logger.Warn("download completed with non-2xx status", "component", "downloader", "url", req.URL, "status_code", resp.StatusCode)
		}
		return result
	}

	if logger != nil {
		logger.Info("download successful", "component", "downloader", "url", req.URL, "status_code", resp.StatusCode, "duration", result.Duration)
	}
	return result
}
