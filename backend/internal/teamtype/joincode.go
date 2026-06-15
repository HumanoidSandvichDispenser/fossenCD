package teamtype

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/psanford/wormhole-william/rendezvous"
	"github.com/psanford/wormhole-william/wordlist"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/nacl/secretbox"
	"salsa.debian.org/vasudev/gospake2"
)

const (
	AppID        = "teamtype"
	DefaultRelay = "ws://relay.magic-wormhole.io:4000/v1"
)

var errMailboxClosed = errors.New("teamtype: rendezvous mailbox channel closed")

// pake phase body: hex(json(pakeMsg)). Mirrors magic-wormhole's PhaseMessage.
type pakeMsg struct {
	PakeV1 string `json:"pake_v1"`
}

// MintJoinCode allocates a wormhole code and, when a peer claims it, hands them
// k's secret address over the encrypted channel. The code is returned right
// away; the returned channel yields the delivery result (nil on success) once a
// peer claims or ctx is done. Pass relay == "" for the public rendezvous server
// (must match the host).
func (k Key) MintJoinCode(ctx context.Context, relay string) (string, <-chan error, error) {
	if relay == "" {
		relay = DefaultRelay
	}

	sideID := randHex(5)
	rc := rendezvous.NewClient(relay, sideID, AppID)
	if _, err := rc.Connect(ctx); err != nil {
		return "", nil, err
	}
	nameplate, err := rc.CreateMailbox(ctx)
	if err != nil {
		return "", nil, err
	}
	code := nameplate + "-" + wordlist.ChooseWords(2)

	done := make(chan error, 1)
	go func() {
		err := deliver(ctx, rc, sideID, code, k.SecretAddress())
		_ = rc.Close(ctx, rendezvous.Happy) // always release the nameplate
		done <- err
		close(done)
	}()

	return code, done, nil
}

// deliver runs the wormhole handshake (PAKE + version) then sends the address as
// the raw phase-"0" message, matching teamtype's Wormhole::send.
func deliver(ctx context.Context, rc *rendezvous.Client, sideID, code, address string) error {
	ch := rc.MsgChan(ctx)
	spake := gospake2.SPAKE2Symmetric(gospake2.NewPassword(code), gospake2.NewIdentityS(AppID))

	body, err := json.Marshal(pakeMsg{PakeV1: hex.EncodeToString(spake.Start())})
	if err != nil {
		return err
	}
	if err := rc.AddMessage(ctx, "pake", hex.EncodeToString(body)); err != nil {
		return err
	}

	_, peerBody, err := readPhase(ch, sideID, "pake")
	if err != nil {
		return err
	}
	peerStart, err := decodePake(peerBody)
	if err != nil {
		return err
	}
	sharedKey, err := spake.Finish(peerStart)
	if err != nil {
		return err
	}

	// version exchange: teamtype's joiner sends one and blocks for ours
	if err := sendEncrypted(ctx, rc, []byte(`{"app_versions":{}}`), sharedKey, sideID, "version"); err != nil {
		return err
	}
	// decrypting a received message uses the SENDER's side
	peerSide, verBody, err := readPhase(ch, sideID, "version")
	if err != nil {
		return err
	}
	if _, err := openEncrypted(verBody, sharedKey, peerSide, "version"); err != nil {
		return err
	}

	return sendEncrypted(ctx, rc, []byte(address), sharedKey, sideID, "0")
}

func decodePake(hexBody string) ([]byte, error) {
	raw, err := hex.DecodeString(hexBody)
	if err != nil {
		return nil, err
	}
	var msg pakeMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	return hex.DecodeString(msg.PakeV1)
}

// readPhase returns the sender side and body of the next peer message for phase.
// The side is needed to derive the decryption key. Relies on magic-wormhole's
// ordering (pake before version) and drops non-matching events.
func readPhase(ch <-chan rendezvous.MailboxEvent, sideID, phase string) (string, string, error) {
	for ev := range ch {
		if ev.Error != nil {
			return "", "", ev.Error
		}
		if ev.Side != sideID && ev.Phase == phase {
			return ev.Side, ev.Body, nil
		}
	}
	return "", "", errMailboxClosed
}

// HKDF-SHA256 over "wormhole:phase:" + sha256(side) + sha256(phase).
func derivePhaseKey(sharedKey []byte, side, phase string) [32]byte {
	sideSha := sha256.Sum256([]byte(side))
	phaseSha := sha256.Sum256([]byte(phase))
	purpose := "wormhole:phase:" + string(sideSha[:]) + string(phaseSha[:])
	r := hkdf.New(sha256.New, sharedKey, nil, []byte(purpose))
	var out [32]byte
	io.ReadFull(r, out[:])
	return out
}

func sendEncrypted(ctx context.Context, rc *rendezvous.Client, plaintext, sharedKey []byte, side, phase string) error {
	box, err := sealPhase(plaintext, sharedKey, side, phase)
	if err != nil {
		return err
	}
	return rc.AddMessage(ctx, phase, hex.EncodeToString(box))
}

// sealPhase returns nonce||ciphertext for a phase message.
func sealPhase(plaintext, sharedKey []byte, side, phase string) ([]byte, error) {
	key := derivePhaseKey(sharedKey, side, phase)
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], plaintext, &nonce, &key), nil
}

func openEncrypted(hexBody string, sharedKey []byte, side, phase string) ([]byte, error) {
	raw, err := hex.DecodeString(hexBody)
	if err != nil {
		return nil, err
	}
	if len(raw) < 24 {
		return nil, fmt.Errorf("teamtype: %q message too short", phase)
	}
	key := derivePhaseKey(sharedKey, side, phase)
	var nonce [24]byte
	copy(nonce[:], raw[:24])
	out, ok := secretbox.Open(nil, raw[24:], &nonce, &key)
	if !ok {
		return nil, fmt.Errorf("teamtype: failed to decrypt %q message", phase)
	}
	return out, nil
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
