package mux

import (
	"context"
	"testing"
	"time"
)

func TestMergeForwardsAllEventsAndCloses(t *testing.T) {
	left := make(chan Event, 2)
	right := make(chan Event, 2)

	left <- Event{EventType: "left-1"}
	left <- Event{EventType: "left-2"}
	right <- Event{EventType: "right-1"}
	right <- Event{EventType: "right-2"}
	close(left)
	close(right)

	out := Merge(context.Background(), nil, left, right)
	seen := make(map[string]bool)
	for event := range out {
		seen[event.EventType] = true
	}

	for _, eventType := range []string{"left-1", "left-2", "right-1", "right-2"} {
		if !seen[eventType] {
			t.Fatalf("missing merged event %q", eventType)
		}
	}
}

func TestMergeClosesOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	blocked := make(chan Event)
	out := Merge(ctx, nil, blocked)

	cancel()

	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("expected merged output to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("merged output did not close after cancellation")
	}
}

func TestMergeWithNoInputsIsClosed(t *testing.T) {
	out := Merge(context.Background(), nil)
	if _, ok := <-out; ok {
		t.Fatal("expected output with no inputs to be closed")
	}
}
