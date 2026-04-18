package demo_test

import (
	"context"
	"testing"
	"time"

	"Concurflow/internal/demo"
)

func TestScenarioBasic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := demo.ScenarioBasic(ctx); err != nil {
		t.Errorf("Basic scenario failed: %v", err)
	}
}

func TestScenarioCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := demo.ScenarioCancellation(ctx); err != nil {
		t.Errorf("Cancellation scenario failed: %v", err)
	}
}

func TestScenarioInvalidURLs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := demo.ScenarioInvalidURLs(ctx); err != nil {
		t.Errorf("Invalid URL scenario failed: %v", err)
	}
}

func TestScenarioMixed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := demo.ScenarioMixed(ctx); err != nil {
		t.Errorf("Mixed scenario failed: %v", err)
	}
}
