package teamtype

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/psanford/wormhole-william/rendezvous"
)

// Cross-impl vectors from magic-wormhole core/key.rs (test_derive_phase_key).
// Matching them proves our phase-key derivation is byte-compatible with the
// teamtype joiner.
func TestDerivePhaseKeyVectors(t *testing.T) {
	mainKey, err := hex.DecodeString("588ba9eef353778b074413a0140205d90d7479e36e0dd4ee35bb729d26131ef1")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		side, phase, want string
	}{
		{"side1", "phase1", "3af6a61d1a111225cc8968c6ca6265efe892065c3ab46de79dda21306b062990"},
		{"side1", "phase2", "88a1dd12182d989ff498022a9656d1e2806f17328d8bf5d8d0c9753e4381a752"},
		{"side2", "phase1", "a306627b436ec23bdae3af8fa90c9ac927780d86be1831003e7f617c518ea689"},
		{"side2", "phase2", "bf99e3e16420f2dad33f9b1ccb0be1462b253d639dacdb50ed9496fa528d8758"},
	}
	for _, c := range cases {
		got := derivePhaseKey(mainKey, c.side, c.phase)
		if hex.EncodeToString(got[:]) != c.want {
			t.Errorf("derivePhaseKey(%s, %s) = %x, want %s", c.side, c.phase, got, c.want)
		}
	}
}

func TestDecodePake(t *testing.T) {
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	body, err := json.Marshal(pakeMsg{PakeV1: hex.EncodeToString(want)})
	if err != nil {
		t.Fatal(err)
	}
	wire := hex.EncodeToString(body)

	got, err := decodePake(wire)
	if err != nil {
		t.Fatalf("decodePake: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("decodePake = %x, want %x", got, want)
	}
}

func TestDecodePakeRejectsGarbage(t *testing.T) {
	if _, err := decodePake("nothex"); err == nil {
		t.Error("expected error for non-hex body")
	}
	if _, err := decodePake(hex.EncodeToString([]byte("not json"))); err == nil {
		t.Error("expected error for non-json body")
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	plaintext := []byte("35bec7a0...#cf629498...") // stand-in secret address

	box, err := sealPhase(plaintext, key, "ab12cd34ef", "0")
	if err != nil {
		t.Fatalf("sealPhase: %v", err)
	}

	got, err := openEncrypted(hex.EncodeToString(box), key, "ab12cd34ef", "0")
	if err != nil {
		t.Fatalf("openEncrypted: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("round-trip = %q, want %q", got, plaintext)
	}
}

func TestOpenEncryptedRejectsTamper(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	box, err := sealPhase([]byte("hello"), key, "side", "0")
	if err != nil {
		t.Fatal(err)
	}
	box[len(box)-1] ^= 0xff // flip a ciphertext byte

	if _, err := openEncrypted(hex.EncodeToString(box), key, "side", "0"); err == nil {
		t.Error("expected decryption failure for tampered ciphertext")
	}
}

func TestOpenEncryptedWrongPhase(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	box, err := sealPhase([]byte("hello"), key, "side", "0")
	if err != nil {
		t.Fatal(err)
	}
	// different phase derives a different key, so the box must not open
	if _, err := openEncrypted(hex.EncodeToString(box), key, "side", "version"); err == nil {
		t.Error("expected decryption failure under a different phase")
	}
}

func TestOpenEncryptedTooShort(t *testing.T) {
	if _, err := openEncrypted(hex.EncodeToString([]byte{1, 2, 3}), nil, "side", "0"); err == nil {
		t.Error("expected error for sub-nonce-length message")
	}
}

func TestReadPhaseMatches(t *testing.T) {
	ch := make(chan rendezvous.MailboxEvent, 3)
	ch <- rendezvous.MailboxEvent{Side: "self", Phase: "pake", Body: "echo"}    // our own, skip
	ch <- rendezvous.MailboxEvent{Side: "peer", Phase: "version", Body: "skip"} // wrong phase, skip
	ch <- rendezvous.MailboxEvent{Side: "peer", Phase: "pake", Body: "want"}
	close(ch)

	side, got, err := readPhase(ch, "self", "pake")
	if err != nil {
		t.Fatalf("readPhase: %v", err)
	}
	if got != "want" {
		t.Fatalf("readPhase body = %q, want %q", got, "want")
	}
	if side != "peer" {
		t.Fatalf("readPhase side = %q, want %q", side, "peer")
	}
}

func TestReadPhasePropagatesError(t *testing.T) {
	sentinel := errors.New("boom")
	ch := make(chan rendezvous.MailboxEvent, 1)
	ch <- rendezvous.MailboxEvent{Error: sentinel}
	close(ch)

	if _, _, err := readPhase(ch, "self", "pake"); !errors.Is(err, sentinel) {
		t.Fatalf("readPhase err = %v, want %v", err, sentinel)
	}
}

func TestReadPhaseClosedChannel(t *testing.T) {
	ch := make(chan rendezvous.MailboxEvent)
	close(ch)

	if _, _, err := readPhase(ch, "self", "pake"); !errors.Is(err, errMailboxClosed) {
		t.Fatalf("readPhase err = %v, want errMailboxClosed", err)
	}
}
