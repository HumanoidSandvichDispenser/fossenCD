// Package teamtype provides utilities for deriving teamtype secret addresses
// from .teamtype/key, which contains the node seed and passphrase. This allows
// us to not have to depend on the teamtype daemon, and a host could easily
// rotate their address by changing the key file (which the secret address is
// derived from). In addition, on the Go side, we could hand off secret
// addresses (and join codes in joincodes.go) through an API without touching
// the teamtype daemon.
package teamtype

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// KeyLen is the on-disk size of .teamtype/key: 32-byte node seed + 32-byte
// passphrase. Must match teamtype's peer.rs::get_keypair.
const KeyLen = 64

// Key is a teamtype keypair loaded from .teamtype/key.
type Key struct {
	// NodeSeed is the ed25519 seed iroh uses as the node SecretKey.
	NodeSeed [32]byte
	// Passphrase is the bearer secret a joiner presents to authenticate.
	Passphrase [32]byte
}

// SecretAddress is the node_id#passphrase string a peer needs to connect.
// node_id is the lowercase-hex ed25519 public key; passphrase is the raw
// 32 bytes hex-encoded (iroh SecretKey Display).
func (k Key) SecretAddress() string {
	return fmt.Sprintf("%s#%s", k.NodeID(), hex.EncodeToString(k.Passphrase[:]))
}

// NodeID is the lowercase-hex iroh node id (ed25519 public key).
func (k Key) NodeID() string {
	pub := ed25519.NewKeyFromSeed(k.NodeSeed[:]).Public().(ed25519.PublicKey)
	return hex.EncodeToString(pub)
}

// LoadKey reads an existing .teamtype/key under baseDir.
func LoadKey(fs afero.Fs, baseDir string) (Key, error) {
	var key Key
	raw, err := afero.ReadFile(fs, keyPath(baseDir))
	if err != nil {
		return key, err
	}
	if len(raw) != KeyLen {
		return key, fmt.Errorf("keyfile is %d bytes, expected %d (teamtype >= 0.7.0)", len(raw), KeyLen)
	}
	copy(key.NodeSeed[:], raw[:32])
	copy(key.Passphrase[:], raw[32:])
	return key, nil
}

// EnsureKey loads .teamtype/key, creating it (0600) if absent. Writing it
// before the daemon starts means the daemon reuses it, so the derived address
// is valid for the hosted session.
func EnsureKey(fs afero.Fs, baseDir string) (Key, error) {
	if key, err := LoadKey(fs, baseDir); err == nil {
		return key, nil
	} else if !os.IsNotExist(err) {
		return key, err
	}

	var key Key
	if _, err := rand.Read(key.NodeSeed[:]); err != nil {
		return key, err
	}
	if _, err := rand.Read(key.Passphrase[:]); err != nil {
		return key, err
	}

	if err := fs.MkdirAll(filepath.Join(baseDir, ".teamtype"), 0o700); err != nil {
		return key, err
	}
	raw := make([]byte, 0, KeyLen)
	raw = append(raw, key.NodeSeed[:]...)
	raw = append(raw, key.Passphrase[:]...)
	if err := afero.WriteFile(fs, keyPath(baseDir), raw, 0o600); err != nil {
		return key, err
	}
	return key, nil
}

func keyPath(baseDir string) string {
	return filepath.Join(baseDir, ".teamtype", "key")
}
