package mux

import (
	"context"
	"log/slog"
	"sync"
)

type Event struct {
	EventType string
	Data      interface{}
}

// Merge combines multiple event channels into a single output channel.
// Returns a channel that receives events from all inputs until context is done or all inputs close.
func Merge(ctx context.Context, logger *slog.Logger, inputs ...<-chan Event) <-chan Event {
	out := make(chan Event, 5)

	go func() {
		defer close(out)

		if len(inputs) == 0 {
			return
		}

		var wg sync.WaitGroup

		// Each input channel gets its own forwarder goroutine
		for _, ch := range inputs {
			wg.Add(1)

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

		// Close output once all forwarders finish
		go func() {
			wg.Wait()
			logger.Debug("all mux inputs exhausted", "component", "mux")
		}()
	}()

	return out
}
