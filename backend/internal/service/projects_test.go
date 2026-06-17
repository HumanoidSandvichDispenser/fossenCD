package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/spf13/afero"
)

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
	keyPath := svc.Projects.shareDir(p.ID) + "/.teamtype/key"
	info, err := svc.Projects.fs.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if info.Size() != 64 {
		t.Fatalf("keyfile is %d bytes, want 64", info.Size())
	}
}

func TestAddressFormatAndStability(t *testing.T) {
	svc := newServices(t)
	ctx := context.Background()
	uid := ownerID(t, svc, "ryan")
	p, _ := svc.Projects.Create(ctx, uid, "thesis")

	addr, err := svc.Projects.Address(ctx, uid, p.ID)
	if err != nil {
		t.Fatalf("address: %v", err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}#[0-9a-f]{64}$`).MatchString(addr) {
		t.Fatalf("address %q not node_id#passphrase hex", addr)
	}

	again, _ := svc.Projects.Address(ctx, uid, p.ID)
	if again != addr {
		t.Fatalf("address not stable: %q vs %q", addr, again)
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
