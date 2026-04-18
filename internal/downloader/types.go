package downloader

import "time"

type DownloadRequest struct {
	ID  int
	URL string
}
type DownloadResult struct {
	ID         int
	URL        string
	StatusCode int
	Err        error
	Duration   time.Duration
}
