package downloader

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/SShogun/Concurflow/internal/config"
)

type Downloader struct {
	cfg    config.Config
	logger *slog.Logger
	client *http.Client
}

type indexedResult struct {
	index  int
	result DownloadResult
}

func New(cfg config.Config, logger *slog.Logger) *Downloader {
	if logger == nil {
		logger = slog.Default()
	}

	return &Downloader{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{},
	}
}

func (d *Downloader) Run(ctx context.Context, reqs []DownloadRequest) ([]DownloadResult, error) {
	if d.cfg.MaxConcurrentDownloads <= 0 {
		return nil, fmt.Errorf("max concurrent downloads must be greater than zero")
	}
	if d.cfg.PerDownloadTimeout <= 0 {
		return nil, fmt.Errorf("per-download timeout must be greater than zero")
	}
	if len(reqs) == 0 {
		d.logger.Info("downloader received empty request list", "component", "downloader")
		return []DownloadResult{}, nil
	}

	d.logger.Info("downloader starting", "component", "downloader", "request_count", len(reqs), "max_concurrent", d.cfg.MaxConcurrentDownloads)

	semaphore := make(chan struct{}, d.cfg.MaxConcurrentDownloads)
	internalResults := make(chan indexedResult, len(reqs))
	var wg sync.WaitGroup

	for i, req := range reqs {
		wg.Add(1)

		go func(idx int, request DownloadRequest) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				d.logger.Info("downloader request canceled before permit acquired", "component", "downloader", "request_id", request.ID, "url", request.URL, "reason", "context_done")
				internalResults <- indexedResult{
					index: idx,
					result: DownloadResult{
						ID:       request.ID,
						URL:      request.URL,
						Err:      ctx.Err(),
						Duration: 0,
					},
				}
				return
			case semaphore <- struct{}{}:
			}

			// A goroutine may release a semaphore slot only after it has acquired one.
			defer func() {
				<-semaphore
			}()

			itemCtx, cancel := context.WithTimeout(ctx, d.cfg.PerDownloadTimeout)
			defer cancel()

			result := fetch(itemCtx, d.client, request, d.logger)
			internalResults <- indexedResult{index: idx, result: result}

			d.logger.Info("downloader request completed", "component", "downloader", "request_id", request.ID, "url", request.URL, "status_code", result.StatusCode, "has_error", result.Err != nil)
		}(i, req)
	}

	go func() {
		wg.Wait()
		close(internalResults)
	}()

	results := make([]DownloadResult, len(reqs))
	for item := range internalResults {
		results[item.index] = item.result
	}

	d.logger.Info("downloader finished", "component", "downloader", "result_count", len(results))
	if err := ctx.Err(); err != nil {
		return results, err
	}
	return results, nil
}
