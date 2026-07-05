// Package config holds server configuration sourced from the environment.
package config

import "os"

type Config struct {
	Addr    string // listen address
	DBPath  string // sqlite file path
	DataDir string // base dir holding per-project share dirs
	EgBin   string // okayeg `eg` daemon binary (on PATH or absolute)
}

// Load reads config from the environment, applying dev defaults.
func Load() Config {
	return Config{
		Addr:    env("FOSSENCD_ADDR", ":8080"),
		DBPath:  env("FOSSENCD_DB", "fossencd.db"),
		DataDir: env("FOSSENCD_DATA_DIR", "data"),
		EgBin:   env("FOSSENCD_EG_BIN", "eg"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}