package teamtype

import (
	"regexp"
	"testing"

	"github.com/spf13/afero"
)

var hexAddr = regexp.MustCompile(`^[0-9a-f]{64}#[0-9a-f]{64}$`)

func TestEnsureKeyCreatesKeyfile(t *testing.T) {
	fs := afero.NewMemMapFs()

	key, err := EnsureKey(fs, "/work")
	if err != nil {
		t.Fatalf("EnsureKey: %v", err)
	}

	raw, err := afero.ReadFile(fs, "/work/.teamtype/key")
	if err != nil {
		t.Fatalf("keyfile not written: %v", err)
	}
	if len(raw) != KeyLen {
		t.Fatalf("keyfile is %d bytes, want %d", len(raw), KeyLen)
	}
	if !hexAddr.MatchString(key.SecretAddress()) {
		t.Fatalf("address %q is not node_id#passphrase hex", key.SecretAddress())
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
	if first.SecretAddress() != second.SecretAddress() {
		t.Fatalf("address changed across calls")
	}
}

func TestSecretAddressDeterministic(t *testing.T) {
	var key Key
	for i := range key.NodeSeed {
		key.NodeSeed[i] = byte(i)
	}
	for i := range key.Passphrase {
		key.Passphrase[i] = byte(0xff - i)
	}

	const want = "03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8" +
		"#fffefdfcfbfaf9f8f7f6f5f4f3f2f1f0efeeedecebeae9e8e7e6e5e4e3e2e1e0"
	if got := key.SecretAddress(); got != want {
		t.Fatalf("SecretAddress mismatch:\n got %q\nwant %q", got, want)
	}
}
