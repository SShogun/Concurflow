package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/SShogun/Concurflow/internal/downloader"
	"github.com/SShogun/Concurflow/internal/logging"
	"github.com/SShogun/Concurflow/internal/pipeline"
)

type App struct {
	cfg    Config
	logger *slog.Logger
}

func New(cfg Config) *App {
	return &App{
		cfg:    cfg,
		logger: logging.New(),
	}
}

// Run orchestrates URL normalization and bounded concurrent downloads.
func (a *App) Run(ctx context.Context) error {
	if err := a.cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, a.cfg.RunTimeout)
	defer cancel()

	a.logger.Info("app starting", "config", fmt.Sprintf("%+v", a.cfg))

	// Sample URLs for demo - in real usage these would come from user input or API.
	urls := []string{
		"https://www.google.com",
		"https://github.com",
		"https://www.wikipedia.org",
		"https://stackoverflow.com",
		"not-a-url",
		"",
	}

	a.logger.Info("phase 1: normalizing urls", "component", "app")
	rawURLs := make([]pipeline.RawURL, len(urls))
	for i, u := range urls {
		rawURLs[i] = pipeline.RawURL{ID: i, URL: u}
	}

	normalized, err := pipeline.RunBuffered(runCtx, a.logger, rawURLs, a.cfg.PipelineBufferSize)
	if err != nil {
		a.logger.Error("pipeline failed", "error", err)
		return err
	}

	a.logger.Info("pipeline complete", "input_count", len(rawURLs), "output_count", len(normalized))

	validURLs := make([]downloader.DownloadRequest, 0, len(normalized))
	for _, nu := range normalized {
		if nu.Valid {
			validURLs = append(validURLs, downloader.DownloadRequest{
				ID:  nu.ID,
				URL: nu.URL,
			})
		}
	}

	a.logger.Info("filtered urls for download", "valid_count", len(validURLs))
	a.logger.Info("phase 2: downloading with rate limit", "component", "app", "max_concurrent", a.cfg.MaxConcurrentDownloads)

	dl := downloader.New(a.cfg, a.logger)
	results, err := dl.Run(runCtx, validURLs)
	if err != nil {
		a.logger.Error("download phase failed", "error", err)
		return err
	}

	a.logger.Info("download phase complete", "total_results", len(results))
	successful := 0
	failed := 0
	for _, r := range results {
		if r.Err == nil && r.StatusCode >= 200 && r.StatusCode < 300 {
			successful++
		} else {
			failed++
		}
	}
	a.logger.Info("download summary", "successful", successful, "failed", failed)

	return nil
}
