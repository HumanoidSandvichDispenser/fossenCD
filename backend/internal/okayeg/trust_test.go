package okayeg

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

const (
	peerA = "01bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"
	peerB = "02bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"
	peerC = "03bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"
)

func readTrustFile(t *testing.T, fs afero.Fs) string {
	t.Helper()
	b, err := afero.ReadFile(fs, filepath.Join("/work", trustRelPath))
	if err != nil {
		t.Fatalf("read trust: %v", err)
	}
	return string(b)
}

func TestSetTrustCreatesFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := SetTrust(fs, "/work", peerA, 7); err != nil {
		t.Fatalf("SetTrust: %v", err)
	}
	want := peerA + " pull push user:7\n"
	if got := readTrustFile(t, fs); got != want {
		t.Fatalf("trust = %q, want %q", got, want)
	}
}

func TestSetTrustReplacesExistingForSamePeer(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = SetTrust(fs, "/work", peerA, 1)
	if err := SetTrust(fs, "/work", peerA, 2); err != nil {
		t.Fatalf("SetTrust: %v", err)
	}
	// only one row for peerA, with the latest user tag, plus a trailing newline
	got := readTrustFile(t, fs)
	if n := strings.Count(got, peerA); n != 1 {
		t.Fatalf("expected 1 row for peerA, got %d in %q", n, got)
	}
	if !strings.Contains(got, peerA+" pull push user:2\n") {
		t.Fatalf("latest grant not written: %q", got)
	}
}

func TestSetTrustPreservesOtherPeersAndComments(t *testing.T) {
	fs := afero.NewMemMapFs()
	// seed the file with a comment and another peer's grant, as a user or `eg
	// trust` might have written
	seed := "# bob's laptop\n" + peerB + " pull push user:3\n"
	if err := afero.WriteFile(fs, "/work/.eg/trust", []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SetTrust(fs, "/work", peerA, 4); err != nil {
		t.Fatalf("SetTrust: %v", err)
	}
	got := readTrustFile(t, fs)
	if !strings.HasPrefix(got, "# bob's laptop\n") {
		t.Fatalf("comment not preserved: %q", got)
	}
	if !strings.Contains(got, peerB+" pull push user:3") {
		t.Fatalf("peerB grant dropped: %q", got)
	}
	if !strings.Contains(got, peerA+" pull push user:4") {
		t.Fatalf("peerA grant missing: %q", got)
	}
}

func TestSetTrustRejectsBadEndpointID(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := SetTrust(fs, "/work", "not-an-id", 1); err == nil {
		t.Fatal("expected error for invalid endpoint id")
	}
}

func TestSetTrustIsIdempotent(t *testing.T) {
	fs := afero.NewMemMapFs()
	for i := 0; i < 3; i++ {
		if err := SetTrust(fs, "/work", peerA, 5); err != nil {
			t.Fatalf("SetTrust %d: %v", i, err)
		}
	}
	got := readTrustFile(t, fs)
	if strings.Count(got, peerA) != 1 {
		t.Fatalf("expected 1 row, got %q", got)
	}
}

func TestRemoveTrust(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = SetTrust(fs, "/work", peerA, 1)
	_ = SetTrust(fs, "/work", peerB, 2)

	if err := RemoveTrust(fs, "/work", peerA); err != nil {
		t.Fatalf("RemoveTrust: %v", err)
	}
	got := readTrustFile(t, fs)
	if strings.Contains(got, peerA) {
		t.Fatalf("peerA not removed: %q", got)
	}
	if !strings.Contains(got, peerB) {
		t.Fatalf("peerB removed: %q", got)
	}
}

func TestRemoveTrustMissingPeerIsNoop(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = SetTrust(fs, "/work", peerA, 1)
	before := readTrustFile(t, fs)
	if err := RemoveTrust(fs, "/work", peerB); err != nil {
		t.Fatalf("RemoveTrust on absent peer: %v", err)
	}
	if after := readTrustFile(t, fs); after != before {
		t.Fatalf("file changed: %q vs %q", after, before)
	}
}

func TestListTrustCollapsesByLastRow(t *testing.T) {
	fs := afero.NewMemMapFs()
	// two rows for peerA: a pull-only grant then the full one. Last wins.
	seed := peerA + " pull\n" + peerA + " pull push user:9\n" + peerB + " pull\n"
	if err := afero.WriteFile(fs, "/work/.eg/trust", []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	grants, err := ListTrust(fs, "/work")
	if err != nil {
		t.Fatalf("ListTrust: %v", err)
	}
	if len(grants) != 2 {
		t.Fatalf("expected 2 grants, got %d", len(grants))
	}
	a := grants[0]
	if a.EndpointID != peerA || !a.Pull || !a.Push || a.UserID != 9 {
		t.Fatalf("peerA grant = %+v, want full push user:9", a)
	}
	b := grants[1]
	if b.EndpointID != peerB || !b.Pull || b.Push {
		t.Fatalf("peerB grant = %+v, want pull-only", b)
	}
}

func TestListTrustSkipsCommentsAndRevoked(t *testing.T) {
	fs := afero.NewMemMapFs()
	seed := "# header\n" + peerA + " pull push revoked\n" + peerB + " pull\n"
	if err := afero.WriteFile(fs, "/work/.eg/trust", []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	grants, _ := ListTrust(fs, "/work")
	// revoked grant is still listed (carries the flag), caller decides to gate
	if len(grants) != 2 {
		t.Fatalf("expected 2 grants incl. revoked, got %d (%+v)", len(grants), grants)
	}
	for _, g := range grants {
		if g.EndpointID == peerA && !g.Revoked {
			t.Errorf("peerA grant not marked revoked: %+v", g)
		}
	}
}
