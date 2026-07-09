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

func TestAddKeyWritesDB(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")

	if err := svc.Keys.AddKey(ctx, a, endpointAlice, "laptop"); err != nil {
		t.Fatalf("AddKey: %v", err)
	}

	keys, err := svc.Keys.ListKeys(ctx, a)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 1 || keys[0].EndpointID != endpointAlice || keys[0].Label != "laptop" {
		t.Fatalf("keys = %+v, want endpointAlice/laptop", keys)
	}
}

// A key is account-scoped, so registering it needs no project at all: it grants
// its owner access to every project they belong to.
func TestAddKeyNeedsNoProject(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")

	if err := svc.Keys.AddKey(ctx, a, endpointAlice, ""); err != nil {
		t.Fatalf("AddKey without a project: %v", err)
	}
}

func TestAddKeyRejectsBadEndpointID(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")

	if err := svc.Keys.AddKey(ctx, a, "nothex", ""); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("want ErrBadRequest, got %v", err)
	}
}

func TestAddKeyIsIdempotent(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")

	for i := 0; i < 3; i++ {
		if err := svc.Keys.AddKey(ctx, a, endpointAlice, "laptop"); err != nil {
			t.Fatalf("AddKey %d: %v", i, err)
		}
	}
	keys, _ := svc.Keys.ListKeys(ctx, a)
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
}

// AuthorizePeer grants an endpoint iff it's a registered key whose owner is a
// member of the project. A key alone (owner not a member) is refused.
func TestAuthorizePeerThroughMembership(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "doc")

	_ = svc.Keys.AddKey(ctx, a, endpointAlice, "")
	_ = svc.Keys.AddKey(ctx, b, endpointBob, "")

	// alice owns the project: her key is authorized
	if ok, err := svc.Projects.AuthorizePeer(ctx, p.ID, endpointAlice); err != nil || !ok {
		t.Fatalf("alice: ok=%v err=%v, want authorized", ok, err)
	}
	// bob has a key but isn't a member: refused
	if ok, _ := svc.Projects.AuthorizePeer(ctx, p.ID, endpointBob); ok {
		t.Fatal("bob authorized without membership")
	}
	// unknown endpoint: refused
	if ok, _ := svc.Projects.AuthorizePeer(ctx, p.ID, endpointCarol); ok {
		t.Fatal("unregistered endpoint authorized")
	}

	// once bob joins, his key is authorized
	if err := svc.Projects.AddMember(ctx, a, p.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	if ok, err := svc.Projects.AuthorizePeer(ctx, p.ID, endpointBob); err != nil || !ok {
		t.Fatalf("bob after join: ok=%v err=%v, want authorized", ok, err)
	}
}

// RemoveKey drops the row and closes the key's live sessions on every project
// its owner belongs to, so revoking a device kills it everywhere at once.
func TestRemoveKeyRevokesAcrossProjects(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	p1, _ := svc.Projects.Create(ctx, a, "one")
	p2, _ := svc.Projects.Create(ctx, a, "two")
	_ = svc.Keys.AddKey(ctx, a, endpointAlice, "laptop")

	if err := svc.Keys.RemoveKey(ctx, a, endpointAlice); err != nil {
		t.Fatalf("RemoveKey: %v", err)
	}

	keys, _ := svc.Keys.ListKeys(ctx, a)
	if len(keys) != 0 {
		t.Fatalf("key survived removal: %+v", keys)
	}
	host := svc.Projects.host.(*noopHost)
	want := map[string]bool{p1.ID + " " + endpointAlice: true, p2.ID + " " + endpointAlice: true}
	if len(host.revoked) != 2 || !want[host.revoked[0]] || !want[host.revoked[1]] {
		t.Fatalf("revoked = %v, want the key closed on both projects", host.revoked)
	}
}

func TestRemoveKeyRejectsOthersKey(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	_ = svc.Keys.AddKey(ctx, a, endpointAlice, "")

	// bob can't delete alice's key: scoped to the caller's own rows
	if err := svc.Keys.RemoveKey(ctx, b, endpointAlice); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// Removing a member keeps their account keys (valid on their other projects)
// but closes any live sessions those keys hold on this project.
func TestRemoveMemberRevokesKeysHere(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	a := ownerID(t, svc, "alice")
	b := ownerID(t, svc, "bob")
	p, _ := svc.Projects.Create(ctx, a, "doc")
	if err := svc.Projects.AddMember(ctx, a, p.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	_ = svc.Keys.AddKey(ctx, b, endpointBob, "laptop")
	_ = svc.Keys.AddKey(ctx, b, endpointCarol, "desktop")

	if err := svc.Projects.RemoveMember(ctx, a, p.ID, b); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	// keys survive: they belong to bob's account, not the project
	keys, _ := svc.Keys.ListKeys(ctx, b)
	if len(keys) != 2 {
		t.Fatalf("bob's keys = %+v, want both kept", keys)
	}
	// but both were revoked on this project
	host := svc.Projects.host.(*noopHost)
	if len(host.revoked) != 2 {
		t.Fatalf("revoked = %v, want both of bob's sessions closed", host.revoked)
	}
}
