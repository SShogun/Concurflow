package downloader

import (
	"Concurflow/internal/config"
	"context"
	"log/slog"
	"net/http"
	"sync"
)

type Downloader struct {
	cfg    config.Config
	logger *slog.Logger
	client *http.Client
}

func New(cfg config.Config, logger *slog.Logger) *Downloader {
	return &Downloader{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{},
	}
}

func (d *Downloader) Run(ctx context.Context, reqs []DownloadRequest) ([]DownloadResult, error) {
	if len(reqs) == 0 {
		d.logger.Info("downloader received empty request list", "component", "downloader")
		return []DownloadResult{}, nil
	}

	d.logger.Info("downloader starting", "component", "downloader", "request_count", len(reqs), "max_concurrent", d.cfg.MaxConcurrentDownloads)

	semaphore := make(chan struct{}, d.cfg.MaxConcurrentDownloads)
	internalResults := make(chan DownloadResult, len(reqs))
	var wg sync.WaitGroup
	var results []DownloadResult
	var resultsMutex sync.Mutex

	for i, req := range reqs {
		wg.Add(1)

		go func(idx int, request DownloadRequest) {
			defer wg.Done()
			defer func() {
				<-semaphore
			}()

			select {
			case <-ctx.Done():
				d.logger.Info("downloader request canceled before permit acquired", "component", "downloader", "request_id", request.ID, "url", request.URL, "reason", "context_done")
				internalResults <- DownloadResult{
					ID:       request.ID,
					URL:      request.URL,
					Err:      ctx.Err(),
					Duration: 0,
				}
				return
			case semaphore <- struct{}{}:
			}

			itemCtx, cancel := context.WithTimeout(ctx, d.cfg.PerDownloadTimeout)
			defer cancel()

			result := fetch(itemCtx, d.client, request, d.logger)
			internalResults <- result

			d.logger.Info("downloader request completed", "component", "downloader", "request_id", request.ID, "url", request.URL, "status_code", result.StatusCode, "has_error", result.Err != nil)
		}(i, req)
	}

	go func() {
		wg.Wait()
		close(internalResults)
	}()

	for result := range internalResults {
		resultsMutex.Lock()
		results = append(results, result)
		resultsMutex.Unlock()
	}

	d.logger.Info("downloader finished", "component", "downloader", "result_count", len(results))
	return results, nil
}
