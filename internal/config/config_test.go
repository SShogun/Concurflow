package config

import "testing"

func TestDefaultConfigIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestValidateRejectsInvalidConcurrencySettings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"worker count", func(cfg *Config) { cfg.WorkerCount = 0 }},
		{"queue depth", func(cfg *Config) { cfg.QueueDepth = -1 }},
		{"pipeline buffer", func(cfg *Config) { cfg.PipelineBufferSize = -1 }},
		{"download concurrency", func(cfg *Config) { cfg.MaxConcurrentDownloads = 0 }},
		{"download timeout", func(cfg *Config) { cfg.PerDownloadTimeout = 0 }},
		{"run timeout", func(cfg *Config) { cfg.RunTimeout = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("expected invalid config to fail validation")
			}
		})
	}
}
