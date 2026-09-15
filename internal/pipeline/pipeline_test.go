package pipeline

import (
	"context"
	"testing"
)

func TestRunClassifiesURLs(t *testing.T) {
	inputs := []RawURL{
		{ID: 1, URL: "   "},
		{ID: 2, URL: "example.com"},
		{ID: 3, URL: "ftp://example.com"},
		{ID: 4, URL: "ht!tp://example.com"},
		{ID: 5, URL: "https:///missing-host"},
		{ID: 6, URL: "https://example.com/path"},
	}

	results, err := Run(context.Background(), nil, inputs)
	if err != nil {
		t.Fatalf("Run returned an unexpected error: %v", err)
	}
	if len(results) != len(inputs) {
		t.Fatalf("expected %d results, got %d", len(inputs), len(results))
	}

	want := []struct {
		valid  bool
		reason Reason
	}{
		{false, empty},
		{false, missingScheme},
		{false, unsupportedScheme},
		{false, malformedURL},
		{false, missingHost},
		{true, fair},
	}

	for i, result := range results {
		if result.Valid != want[i].valid {
			t.Fatalf("result %d validity = %v, want %v", i, result.Valid, want[i].valid)
		}
		if len(result.Reason) != 1 || result.Reason[0] != want[i].reason {
			t.Fatalf("result %d reason = %v, want %q", i, result.Reason, want[i].reason)
		}
	}
}

func TestTransformStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan RawURL)
	out := Transform(ctx, in, nil)

	cancel()
	if _, ok := <-out; ok {
		t.Fatal("expected transform output to close after cancellation")
	}
}

func BenchmarkNormalizeValidURL(b *testing.B) {
	raw := RawURL{ID: 1, URL: " https://example.com/a/path?x=1 "}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := normalize(raw, nil)
		if !result.Valid {
			b.Fatal("expected URL to remain valid")
		}
	}
}
