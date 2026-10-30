package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type recordingTransport struct {
	mu   sync.Mutex
	urls []string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.urls = append(r.urls, req.URL.String())
	r.mu.Unlock()
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok"))}, nil
}
func TestRunFiltersInvalidURLsBeforeHTTP(t *testing.T) {
	transport := &recordingTransport{}
	previous := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = previous })
	urls := []string{"", "example.com", "ftp://example.test/file", "http://%zz", "https:///missing-host", "http://:80", "http://example.test:99999", "http://example.test/valid"}
	if err := New(DefaultConfig()).run(context.Background(), urls); err != nil {
		t.Fatal(err)
	}
	transport.mu.Lock()
	defer transport.mu.Unlock()
	if len(transport.urls) != 1 || transport.urls[0] != "http://example.test/valid" {
		t.Fatalf("HTTP requests = %v", transport.urls)
	}
}
