package okayeg

import (
	"encoding/hex"
	"regexp"
	"testing"

	"github.com/spf13/afero"
)

var hexID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestEnsureKeyCreatesKeyfile(t *testing.T) {
	fs := afero.NewMemMapFs()

	key, err := EnsureKey(fs, "/work")
	if err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}

	raw, err := afero.ReadFile(fs, "/work/.eg/key")
	if err != nil {
		t.Fatalf("keyfile not written: %v", err)
	}
	if len(raw) != KeyLen {
		t.Fatalf("keyfile is %d bytes, want %d", len(raw), KeyLen)
	}
	if !hexID.MatchString(key.EndpointID()) {
		t.Fatalf("endpoint id %q is not 64 lowercase hex chars", key.EndpointID())
	}
}

func TestEnsureKeyIsIdempotent(t *testing.T) {
	fs := afero.NewMemMapFs()

	first, err := EnsureKey(fs, "/work")
	if err != nil {
		t.Fatalf("first EnsureKey: %v", err)
	}
	second, err := EnsureKey(fs, "/work")
	if err != nil {
		t.Fatalf("second EnsureKey: %v", err)
	}

	if first != second {
		t.Fatalf("EnsureKey regenerated the key: %x != %x", first, second)
	}
	if first.EndpointID() != second.EndpointID() {
		t.Fatalf("endpoint id changed across calls")
	}
}

// EndpointIDDerivation deterministically derives the public id from a seed and
// matches `eg id` from the same .eg/key bytes (verified against the
// built eg binary). The vector below is the one eg produced for this seed, so
// any change to the derivation (e.g. someone reaching for base32) fails.
func TestEndpointIDDerivation(t *testing.T) {
	var key Key
	for i := range key.Seed {
		key.Seed[i] = byte(i)
	}
	// eg id over .eg/key = 000102...1f produced this lowercase-hex public key
	// (the same id teamtype derived from the same seed, confirming format parity).
	const want = "03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8"
	if got := key.EndpointID(); got != want {
		t.Fatalf("EndpointID = %q, want %q (eg id format drift)", got, want)
	}
}

// Confirmed against `eg id` end to end: load the seed eg wrote, derive the id,
// and it must equal what `eg id` printed.
func TestEndpointIDMatchesEgId(t *testing.T) {
	const seedHex = "a77a71e4f5d23f3ca0214b1061375cff71cb7c0fcd1b834773f41914e555310d"
	const egIDSaid = "01bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"

	fs := afero.NewMemMapFs()
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "/work/.eg/key", seed, 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := LoadKey(fs, "/work")
	if err != nil {
		t.Fatalf("LoadKey: %v", err)
	}
	if got := key.EndpointID(); got != egIDSaid {
		t.Fatalf("derived id %q != eg id output %q", got, egIDSaid)
	}
}

func TestValidEndpointID(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"01bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b", true},
		{"01BC8405C2648427256E61FC6E3E52236635F87460EEB44076CCB361C601C74B", false}, // uppercase
		{"abc", false}, // too short
		{"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", false}, // not hex
		{"", false},
	}
	for _, c := range cases {
		if got := ValidEndpointID(c.s); got != c.want {
			t.Errorf("ValidEndpointID(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}
