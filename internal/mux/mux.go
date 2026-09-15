package mux

import (
	"context"
	"log/slog"
	"sync"
)

type Event struct {
	EventType string
	Data      any
}

// Merge combines multiple event channels into a single output channel.
// The output closes only after every input is exhausted or cancellation stops all forwarders.
func Merge(ctx context.Context, logger *slog.Logger, inputs ...<-chan Event) <-chan Event {
	out := make(chan Event, 5)

	if len(inputs) == 0 {
		close(out)
		return out
	}

	var wg sync.WaitGroup
	wg.Add(len(inputs))

	for _, ch := range inputs {
		go func(in <-chan Event) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case evt, ok := <-in:
					if !ok {
						return
					}

					select {
					case out <- evt:
					case <-ctx.Done():
						return
					}
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		if logger != nil {
			logger.Debug("all mux inputs exhausted", "component", "mux")
		}
		close(out)
	}()

	return out
}
