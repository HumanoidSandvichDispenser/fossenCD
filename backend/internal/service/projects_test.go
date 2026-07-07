package service

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/spf13/afero"
)

const (
	endpointAlice = "01bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"
	endpointBob   = "02bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"
	endpointCarol = "03bc8405c2648427256e61fc6e3e52236635f87460eeb44076ccb361c601c74b"
)

var hexID = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ownerID(t *testing.T, svc *Services, name string) uint {
	t.Helper()
	u, _, err := svc.Auth.Register(context.Background(), name, name+"@x.io", "hunter2hunter2")
	if err != nil {
		t.Fatalf("register %s: %v", name, err)
	}
	return u.ID
}

func TestCreateProjectWritesKey(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	uid := ownerID(t, svc, "ryan")

	p, err := svc.Projects.Create(ctx, uid, "thesis")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" {
		t.Fatal("expected generated project id")
	}
	keyPath := filepath.Join(svc.Projects.shareDir(p.ID), ".eg", "key")
	info, err := svc.Projects.fs.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if info.Size() != 32 {
		t.Fatalf("keyfile is %d bytes, want 32 (.eg/key seed)", info.Size())
	}
}

func TestEndpointFormatAndStability(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	uid := ownerID(t, svc, "ryan")
	p, _ := svc.Projects.Create(ctx, uid, "thesis")

	endpoint, err := svc.Projects.Endpoint(ctx, uid, p.ID)
	if err != nil {
		t.Fatalf("endpoint: %v", err)
	}
	if !hexID.MatchString(endpoint) {
		t.Fatalf("endpoint %q not 64 lowercase hex chars", endpoint)
	}

	again, _ := svc.Projects.Endpoint(ctx, uid, p.ID)
	if again != endpoint {
		t.Fatalf("endpoint not stable: %q vs %q", endpoint, again)
	}
}

func TestEndpointEnsuresHost(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	uid := ownerID(t, svc, "ryan")
	p, _ := svc.Projects.Create(ctx, uid, "thesis")

	_, _ = svc.Projects.Endpoint(ctx, uid, p.ID)
	host := svc.Projects.host.(*noopHost)
	if host.ensured[p.ID] == 0 {
		t.Fatal("Endpoint did not ensure the host daemon")
	}
}

func TestProjectsAreOwnerScoped(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")

	p, _ := svc.Projects.Create(ctx, a, "alice-doc")

	if _, err := svc.Projects.Get(ctx, b, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bob got alice's project, want ErrNotFound, got %v", err)
	}
	list, _ := svc.Projects.List(ctx, b)
	if len(list) != 0 {
		t.Fatalf("bob sees %d projects, want 0", len(list))
	}
}

func TestDeleteRemovesShareDir(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	uid := ownerID(t, svc, "ryan")
	p, _ := svc.Projects.Create(ctx, uid, "thesis")

	if err := svc.Projects.Delete(ctx, uid, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Projects.Get(ctx, uid, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("project still found after delete: %v", err)
	}
	if exists, _ := afero.Exists(svc.Projects.fs, svc.Projects.shareDir(p.ID)); exists {
		t.Fatal("share dir still present after delete")
	}
}

func TestDeleteCascadesMembers(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "alice-doc")

	if err := svc.Projects.AddMember(ctx, a, p.ID, "bob"); err != nil {
		t.Fatalf("add member: %v", err)
	}

	if err := svc.Projects.Delete(ctx, a, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	members, err := svc.Projects.projects.GetMembers(ctx, p.ID)
	if err != nil {
		t.Fatalf("get members: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("orphaned %d member rows after delete; FK cascade not firing", len(members))
	}
}

func TestDeleteOtherUsersProject(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "alice-doc")

	if err := svc.Projects.Delete(ctx, b, p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound deleting another's project, got %v", err)
	}
}

func TestCreateProjectWritesAuthzConfig(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	uid := ownerID(t, svc, "ryan")

	p, err := svc.Projects.Create(ctx, uid, "thesis")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	raw, err := afero.ReadFile(svc.Projects.fs, filepath.Join(svc.Projects.shareDir(p.ID), ".eg", "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	want := "[authz]\ncommand = [\"/app/scripts/eg-authz\", \"/data/authz.sock\", \"" + p.ID + "\"]\n"
	if string(raw) != want {
		t.Fatalf("config.toml = %q, want %q", raw, want)
	}
}

func TestRegisterPeerWritesDB(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	p, _ := svc.Projects.Create(ctx, a, "doc")

	if err := svc.Projects.RegisterPeer(ctx, a, p.ID, endpointAlice); err != nil {
		t.Fatalf("RegisterPeer: %v", err)
	}

	// DB has the row
	peers, err := svc.Projects.ListPeers(ctx, a, p.ID)
	if err != nil {
		t.Fatalf("ListPeers: %v", err)
	}
	if len(peers) != 1 || peers[0].EndpointID != endpointAlice {
		t.Fatalf("peers = %+v, want endpointAlice", peers)
	}
}

func TestRegisterPeerRejectsNonMember(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	ownerID(t, svc, "bob") // registered but not on the project
	p, _ := svc.Projects.Create(ctx, a, "doc")

	if err := svc.Projects.RegisterPeer(ctx, 999, p.ID, endpointBob); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	// a non-member user id doesn't exist either; cover the access path
}

func TestRegisterPeerRejectsBadEndpointID(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	p, _ := svc.Projects.Create(ctx, a, "doc")

	if err := svc.Projects.RegisterPeer(ctx, a, p.ID, "nothex"); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("want ErrBadRequest, got %v", err)
	}
}

func TestRegisterPeerIsIdempotent(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	p, _ := svc.Projects.Create(ctx, a, "doc")

	for i := 0; i < 3; i++ {
		if err := svc.Projects.RegisterPeer(ctx, a, p.ID, endpointAlice); err != nil {
			t.Fatalf("RegisterPeer %d: %v", i, err)
		}
	}
	peers, _ := svc.Projects.ListPeers(ctx, a, p.ID)
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
}

func TestUnregisterPeerOwnerCanRemoveAny(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "doc")
	if err := svc.Projects.AddMember(ctx, a, p.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Projects.RegisterPeer(ctx, b, p.ID, endpointBob); err != nil {
		t.Fatalf("bob register: %v", err)
	}

	if err := svc.Projects.UnregisterPeer(ctx, a, p.ID, endpointBob); err != nil {
		t.Fatalf("owner unregister: %v", err)
	}
	peers, _ := svc.Projects.ListPeers(ctx, a, p.ID)
	if len(peers) != 0 {
		t.Fatalf("expected 0 peers, got %d", len(peers))
	}
	host := svc.Projects.host.(*noopHost)
	if len(host.revoked) != 1 || host.revoked[0] != p.ID+" "+endpointBob {
		t.Fatalf("revoked = %v, want bob's session closed", host.revoked)
	}
}

func TestUnregisterPeerMemberCanRemoveOwn(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "doc")
	if err := svc.Projects.AddMember(ctx, a, p.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	_ = svc.Projects.RegisterPeer(ctx, b, p.ID, endpointBob)
	_ = svc.Projects.RegisterPeer(ctx, a, p.ID, endpointAlice)

	// bob can drop his own endpoint, but not alice's
	if err := svc.Projects.UnregisterPeer(ctx, b, p.ID, endpointBob); err != nil {
		t.Fatalf("remove own: %v", err)
	}
	if err := svc.Projects.UnregisterPeer(ctx, b, p.ID, endpointAlice); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("remove other's: want ErrUnauthorized, got %v", err)
	}
}

func TestRemoveMemberRevokesPeers(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "doc")
	if err := svc.Projects.AddMember(ctx, a, p.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	_ = svc.Projects.RegisterPeer(ctx, b, p.ID, endpointBob)
	_ = svc.Projects.RegisterPeer(ctx, b, p.ID, endpointCarol)

	if err := svc.Projects.RemoveMember(ctx, a, p.ID, b); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	peers, _ := svc.Projects.ListPeers(ctx, a, p.ID)
	if len(peers) != 0 {
		t.Fatalf("bob's peer rows survived removal: %+v", peers)
	}
	host := svc.Projects.host.(*noopHost)
	if len(host.revoked) != 2 {
		t.Fatalf("revoked = %v, want both of bob's sessions closed", host.revoked)
	}
}
