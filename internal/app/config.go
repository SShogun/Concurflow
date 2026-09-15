package app

import "github.com/SShogun/Concurflow/internal/config"

// Config aliases the central runtime configuration.
type Config = config.Config

func DefaultConfig() Config {
	return config.Default()
}
