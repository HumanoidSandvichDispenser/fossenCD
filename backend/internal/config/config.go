// Package config holds server configuration sourced from the environment.
package config

import "os"

type Config struct {
	Addr          string // listen address
	DBPath        string // sqlite file path
	DataDir       string // base dir holding per-project share dirs
	Relay         string // wormhole rendezvous url ("" = teamtype default)
	TeamtypeBin   string // teamtype daemon binary (on PATH or absolute)
	IrohRelay     string // host --iroh-relay ("" = teamtype default)
	IrohPkarr     string // host --iroh-pkarr-relay
	IrohDNSDomain string // host --iroh-dns-domain
}

// Load reads config from the environment, applying dev defaults.
func Load() Config {
	return Config{
		Addr:          env("FOSSENCD_ADDR", ":8080"),
		DBPath:        env("FOSSENCD_DB", "fossencd.db"),
		DataDir:       env("FOSSENCD_DATA_DIR", "data"),
		Relay:         os.Getenv("FOSSENCD_RELAY"),
		TeamtypeBin:   env("FOSSENCD_TEAMTYPE_BIN", "teamtype"),
		IrohRelay:     os.Getenv("FOSSENCD_IROH_RELAY"),
		IrohPkarr:     os.Getenv("FOSSENCD_IROH_PKARR_RELAY"),
		IrohDNSDomain: os.Getenv("FOSSENCD_IROH_DNS_DOMAIN"),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
