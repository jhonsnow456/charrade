package config

import (
	"os"
	"time"
)

type Config struct {
	Addr           string
	RoundDuration  time.Duration
	NextRoundDelay time.Duration
}

func Default() Config {
	return Config{
		Addr:           ":8080",
		RoundDuration:  60 * time.Second,
		NextRoundDelay: 4 * time.Second,
	}
}

func Load() Config {
	cfg := Default()
	if v := os.Getenv("ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("ROUND_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RoundDuration = d
		}
	}
	if v := os.Getenv("NEXT_ROUND_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.NextRoundDelay = d
		}
	}
	return cfg
}
