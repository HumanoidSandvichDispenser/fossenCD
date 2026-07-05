// Package okayeg bridges fossenCD to the okayeg `eg` daemon. It derives the
// host's EndpointId from .eg/key, manages the per-repo .eg/trust file that
// `eg serve` re-reads on every incoming connection, and supervises an `eg serve`
// process per project.
//
// The on-disk shapes match eg::net (KEY_PATH = ".eg/key", a single 32-byte
// ed25519 seed) and eg::trust (TRUST_PATH = ".eg/trust"). The EndpointId string
// format is iroh 1.0's EndpointId Display, which is lowercase hex of the ed25519
// public key, the same derivation teamtype used for its node id.
package okayeg

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
)

// KeyLen is the on-disk size of .eg/key: a single 32-byte ed25519 seed. Must
// match eg::net::repo_secret (".eg/key").
const KeyLen = 32

// endpointIDRe matches an iroh EndpointId as eg emits and accepts one:
// 64 lowercase hex chars (the ed25519 public key). The browser wasm client's
// nodeId() uses the same Display, so registrations always come in this form.
var endpointIDRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Key is an okayeg host keypair loaded from .eg/key: an ed25519 seed.
type Key struct {
	Seed [KeyLen]byte
}

// EndpointID is the lowercase-hex ed25519 public key a peer dials (with `eg
// join`/`eg pull`) and that the host authorizes in .eg/trust. It matches iroh
// 1.0's EndpointId Display and eg's `eg id` output, byte for byte: verified
// against `eg id` from the same seed.
func (k Key) EndpointID() string {
	pub := ed25519.NewKeyFromSeed(k.Seed[:]).Public().(ed25519.PublicKey)
	return hex.EncodeToString(pub)
}

// ValidEndpointID reports whether s is a well-formed EndpointId as the browser
// sends it (64 lowercase hex chars). Used to reject bogus registrations before
// they reach the trust file.
func ValidEndpointID(s string) bool {
	return endpointIDRe.MatchString(s)
}

// LoadKey reads an existing .eg/key under baseDir.
func LoadKey(fs afero.Fs, baseDir string) (Key, error) {
	var key Key

	raw, err := afero.ReadFile(fs, keyPath(baseDir))
	if err != nil {
		return key, err
	}
	if len(raw) != KeyLen {
		return key, fmt.Errorf("keyfile is %d bytes, expected %d (.eg/key)", len(raw), KeyLen)
	}

	copy(key.Seed[:], raw)
	return key, nil
}

// EnsureKey loads .eg/key, creating it (0600, with .eg/ 0700) if absent.
// Writing it before the daemon starts gives the repo a stable identity, so the
// derived EndpointId is valid for the hosted session.
func EnsureKey(fs afero.Fs, baseDir string) (Key, error) {
	if key, err := LoadKey(fs, baseDir); err == nil {
		return key, nil
	} else if !os.IsNotExist(err) {
		return key, err
	}

	var key Key
	if _, err := rand.Read(key.Seed[:]); err != nil {
		return key, err
	}
	if err := fs.MkdirAll(filepath.Join(baseDir, egDir), 0o700); err != nil {
		return key, err
	}
	if err := afero.WriteFile(fs, keyPath(baseDir), key.Seed[:], 0o600); err != nil {
		return key, err
	}

	return key, nil
}

func keyPath(baseDir string) string {
	return filepath.Join(baseDir, egDir, "key")
}
