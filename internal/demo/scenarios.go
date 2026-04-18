package demo

import (
	"context"
	"fmt"
	"time"

	"Concurflow/internal/app"
	"Concurflow/internal/downloader"
	"Concurflow/internal/logging"
	"Concurflow/internal/pipeline"
)

// ScenarioBasic runs a simple URL normalization and download flow
func ScenarioBasic(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Basic Flow ===")

	cfg := app.DefaultConfig()
	a := app.New(cfg)

	return a.Run(ctx)
}

// ScenarioCancellation tests that cancellation properly stops all subsystems
func ScenarioCancellation(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Cancellation ===")

	cfg := app.DefaultConfig()
	cfg.MaxConcurrentDownloads = 2

	// Many URLs to ensure we can cancel mid-flight
	urls := make([]string, 20)
	for i := 0; i < 20; i++ {
		urls[i] = fmt.Sprintf("https://httpbin.org/delay/%d", (i%3)+1)
	}

	rawURLs := make([]pipeline.RawURL, len(urls))
	for i, u := range urls {
		rawURLs[i] = pipeline.RawURL{ID: i, URL: u}
	}

	// Create a short-lived context
	shortCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	logger.Info("running pipeline with short timeout", "urls", len(urls), "timeout_sec", 3)

	normalized, err := pipeline.Run(shortCtx, logger, rawURLs)
	if err != nil {
		logger.Info("expected cancellation/timeout in pipeline", "error", err)
	} else {
		logger.Info("pipeline results", "count", len(normalized))

		validURLs := make([]downloader.DownloadRequest, 0)
		for _, nu := range normalized {
			if nu.Valid {
				validURLs = append(validURLs, downloader.DownloadRequest{ID: nu.ID, URL: nu.URL})
			}
		}

		dl := downloader.New(cfg, logger)
		dlCtx, dlCancel := context.WithTimeout(shortCtx, 2*time.Second)
		defer dlCancel()

		results, _ := dl.Run(dlCtx, validURLs)
		logger.Info("download results despite timeout", "count", len(results))
	}

	return nil
}

// ScenarioBackpressure tests that the downloader respects MaxConcurrentDownloads
func ScenarioBackpressure(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Backpressure (Rate Limiting) ===")

	cfg := app.DefaultConfig()
	cfg.MaxConcurrentDownloads = 2 // Very restrictive for demo
	cfg.PerDownloadTimeout = 5 * time.Second

	// URLs that have observable delays
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
	logger.Info("starting backpressure test", "max_concurrent", cfg.MaxConcurrentDownloads, "urls", len(urls))

	results, err := dl.Run(runCtx, requests)

	elapsed := time.Since(start)
	logger.Info("backpressure test complete", "duration_sec", elapsed.Seconds(), "results", len(results), "error", err)

	// With max 2 concurrent and 5 urls with 1 sec each, should take ~2.5 seconds minimum
	if elapsed < 2*time.Second {
		logger.Warn("completed too quickly, backpressure may not be enforced", "elapsed_sec", elapsed.Seconds())
	}

	return nil
}

// ScenarioInvalidURLs tests handling of malformed URLs
func ScenarioInvalidURLs(ctx context.Context) error {
	logger := logging.New()
	logger.Info("=== SCENARIO: Invalid URLs ===")

	badURLs := []string{
		"not-a-url",
		"",
		"ftp://unsupported.example.com",
		"ht!tp://weird.example.com",
		"   ",
	}

	rawURLs := make([]pipeline.RawURL, len(badURLs))
	for i, u := range badURLs {
		rawURLs[i] = pipeline.RawURL{ID: i, URL: u}
	}

	pCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	logger.Info("normalizing invalid urls", "count", len(badURLs))
	normalized, _ := pipeline.Run(pCtx, logger, rawURLs)

	validCount := 0
	invalidCount := 0
	for _, n := range normalized {
		if n.Valid {
			validCount++
		} else {
			invalidCount++
			logger.Info("invalid url details", "id", n.ID, "url", n.URL, "reasons", len(n.Reason))
		}
	}

	logger.Info("invalid url test complete", "valid", validCount, "invalid", invalidCount)
	return nil
}

// ScenarioMixed runs with a realistic mix of valid and invalid URLs
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

	logger.Info("starting mixed scenario", "total_urls", len(urls))
	normalized, _ := pipeline.Run(pCtx, logger, rawURLs)

	validURLs := make([]downloader.DownloadRequest, 0)
	for _, n := range normalized {
		if n.Valid {
			validURLs = append(validURLs, downloader.DownloadRequest{ID: n.ID, URL: n.URL})
		}
	}

	logger.Info("after pipeline filtering", "valid_for_download", len(validURLs))

	if len(validURLs) > 0 {
		dl := downloader.New(cfg, logger)
		dlCtx, dlCancel := context.WithTimeout(pCtx, 8*time.Second)
		defer dlCancel()

		results, _ := dl.Run(dlCtx, validURLs)
		logger.Info("download results", "count", len(results))
	}

	return nil
}
