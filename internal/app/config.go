package app

// Re-export Config and DefaultConfig for backward compatibility
import "Concurflow/internal/config"

type Config = config.Config

func DefaultConfig() Config {
	return config.Default()
}
