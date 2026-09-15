package demo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SShogun/Concurflow/internal/app"
	"github.com/SShogun/Concurflow/internal/downloader"
	"github.com/SShogun/Concurflow/internal/logging"
	"github.com/SShogun/Concurflow/internal/pipeline"
)

// ScenarioBasic runs a simple URL normalization and download flow.
func ScenarioBasic(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Basic Flow ===")

	cfg := app.DefaultConfig()
	a := app.New(cfg)

	return a.Run(ctx)
}

// ScenarioCancellation verifies that cancellation stops downloader work cleanly.
func ScenarioCancellation(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Cancellation ===")

	cfg := app.DefaultConfig()
	cfg.MaxConcurrentDownloads = 2

	urls := make([]string, 20)
	for i := 0; i < 20; i++ {
		urls[i] = fmt.Sprintf("https://httpbin.org/delay/%d", (i%3)+1)
	}

	rawURLs := make([]pipeline.RawURL, len(urls))
	for i, u := range urls {
		rawURLs[i] = pipeline.RawURL{ID: i, URL: u}
	}

	shortCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	normalized, err := pipeline.RunBuffered(shortCtx, logger, rawURLs, cfg.PipelineBufferSize)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Info("pipeline canceled as expected", "error", err)
			return nil
		}
		return fmt.Errorf("run cancellation pipeline: %w", err)
	}

	validURLs := make([]downloader.DownloadRequest, 0, len(normalized))
	for _, nu := range normalized {
		if nu.Valid {
			validURLs = append(validURLs, downloader.DownloadRequest{ID: nu.ID, URL: nu.URL})
		}
	}

	dl := downloader.New(cfg, logger)
	dlCtx, dlCancel := context.WithTimeout(shortCtx, 2*time.Second)
	defer dlCancel()

	results, err := dl.Run(dlCtx, validURLs)
	logger.Info("download cancellation results", "count", len(results), "error", err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("run cancellation downloader: %w", err)
	}
	return fmt.Errorf("expected downloader cancellation but run completed")
}

// ScenarioBackpressure demonstrates that MaxConcurrentDownloads bounds network work.
func ScenarioBackpressure(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Backpressure (Rate Limiting) ===")

	cfg := app.DefaultConfig()
	cfg.MaxConcurrentDownloads = 2
	cfg.PerDownloadTimeout = 5 * time.Second

	urls := []string{
		"https://httpbin.org/delay/1",
		"https://httpbin.org/delay/1",
		"https://httpbin.org/delay/1",
		"https://httpbin.org/delay/1",
		"https://httpbin.org/delay/1",
	}

	requests := make([]downloader.DownloadRequest, len(urls))
	for i, u := range urls {
		requests[i] = downloader.DownloadRequest{ID: i, URL: u}
	}

	dl := downloader.New(cfg, logger)
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	start := time.Now()
	results, err := dl.Run(runCtx, requests)
	elapsed := time.Since(start)
	logger.Info("backpressure scenario complete", "duration_sec", elapsed.Seconds(), "results", len(results), "error", err)
	if err != nil {
		return fmt.Errorf("run backpressure downloader: %w", err)
	}
	return nil
}

// ScenarioInvalidURLs tests handling of malformed URLs without network I/O.
func ScenarioInvalidURLs(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Invalid URLs ===")

	cfg := app.DefaultConfig()
	badURLs := []string{
		"not-a-url",
		"",
		"ftp://unsupported.example.com",
		"ht!tp://weird.example.com",
		"https:///missing-host",
		"   ",
	}

	rawURLs := make([]pipeline.RawURL, len(badURLs))
	for i, u := range badURLs {
		rawURLs[i] = pipeline.RawURL{ID: i, URL: u}
	}

	pCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	normalized, err := pipeline.RunBuffered(pCtx, logger, rawURLs, cfg.PipelineBufferSize)
	if err != nil {
		return fmt.Errorf("normalize invalid URL scenario: %w", err)
	}

	validCount := 0
	invalidCount := 0
	for _, n := range normalized {
		if n.Valid {
			validCount++
		} else {
			invalidCount++
			logger.Info("invalid url details", "id", n.ID, "url", n.URL, "reasons", n.Reason)
		}
	}

	logger.Info("invalid url scenario complete", "valid", validCount, "invalid", invalidCount)
	if validCount != 0 || invalidCount != len(badURLs) {
		return fmt.Errorf("unexpected validation counts: valid=%d invalid=%d", validCount, invalidCount)
	}
	return nil
}

// ScenarioMixed runs with a realistic mix of valid and invalid URLs.
func ScenarioMixed(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Mixed Valid/Invalid ===")

	cfg := app.DefaultConfig()
	cfg.MaxConcurrentDownloads = 3

	urls := []string{
		"https://www.example.com",
		"invalid",
		"https://httpbin.org/status/200",
		"",
		"https://httpbin.org/status/404",
		"not a url at all",
		"https://httpbin.org/status/500",
	}

	rawURLs := make([]pipeline.RawURL, len(urls))
	for i, u := range urls {
		rawURLs[i] = pipeline.RawURL{ID: i, URL: u}
	}

	pCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	normalized, err := pipeline.RunBuffered(pCtx, logger, rawURLs, cfg.PipelineBufferSize)
	if err != nil {
		return fmt.Errorf("run mixed pipeline: %w", err)
	}

	validURLs := make([]downloader.DownloadRequest, 0, len(normalized))
	for _, n := range normalized {
		if n.Valid {
			validURLs = append(validURLs, downloader.DownloadRequest{ID: n.ID, URL: n.URL})
		}
	}

	logger.Info("after pipeline filtering", "valid_for_download", len(validURLs))
	if len(validURLs) == 0 {
		return nil
	}

	dl := downloader.New(cfg, logger)
	dlCtx, dlCancel := context.WithTimeout(pCtx, 8*time.Second)
	defer dlCancel()

	results, err := dl.Run(dlCtx, validURLs)
	logger.Info("download results", "count", len(results), "error", err)
	if err != nil {
		return fmt.Errorf("run mixed downloader: %w", err)
	}
	return nil
}
