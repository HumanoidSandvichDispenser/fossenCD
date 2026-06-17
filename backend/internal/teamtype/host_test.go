package teamtype

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition not met within %s", d)
}

func newManager(t *testing.T, backoff time.Duration) *HostManager {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewHostManager(ctx, HostOptions{RestartBackoff: backoff})
}

// fakeRun counts starts and, per call, blocks until either its ctx is cancelled
// (a long-running daemon stopped via SIGTERM) or block is closed (a crash).
type fakeRun struct {
	mu     sync.Mutex
	starts int
	block  chan struct{}
}

func (f *fakeRun) run(ctx context.Context, _ string) error {
	f.mu.Lock()
	f.starts++
	f.mu.Unlock()
	select {
	case <-ctx.Done():
	case <-f.block:
	}
	return nil
}

func (f *fakeRun) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.starts
}

func TestEnsureHostIdempotent(t *testing.T) {
	m := newManager(t, time.Second)
	f := &fakeRun{block: make(chan struct{})}
	m.run = f.run

	m.EnsureHost("a")
	m.EnsureHost("a")
	waitFor(t, time.Second, func() bool { return f.count() == 1 })

	// a second daemon would bump the count
	time.Sleep(20 * time.Millisecond)
	if n := f.count(); n != 1 {
		t.Fatalf("expected exactly 1 start, got %d", n)
	}
}

func TestRestartOnCrash(t *testing.T) {
	m := newManager(t, time.Millisecond)
	f := &fakeRun{block: make(chan struct{})}
	close(f.block) // every run returns immediately
	m.run = f.run

	m.EnsureHost("a")
	waitFor(t, time.Second, func() bool { return f.count() >= 3 })
}

func TestStopTerminatesAndWaits(t *testing.T) {
	m := newManager(t, time.Millisecond)
	f := &fakeRun{block: make(chan struct{})}
	m.run = f.run

	m.EnsureHost("a")
	waitFor(t, time.Second, func() bool { return f.count() == 1 })

	m.Stop("a") // cancels ctx; fakeRun returns, supervise must not restart

	if n := f.count(); n != 1 {
		t.Fatalf("expected no restart after Stop, got %d starts", n)
	}
	m.mu.Lock()
	_, tracked := m.hosts["a"]
	m.mu.Unlock()
	if tracked {
		t.Fatal("host still tracked after Stop")
	}
}

func TestStopUnknownIsNoop(t *testing.T) {
	m := newManager(t, time.Millisecond)
	m.Stop("nope") // must not block or panic
}

// execRun must clear a leftover socket before launching, else `teamtype share`
// prompts (y/N) on stdin we never connect and the supervisor crash-loops.
func TestExecRunRemovesStaleSocket(t *testing.T) {
	dir := t.TempDir()
	m := NewHostManager(context.Background(), HostOptions{Bin: "true", DataDir: dir})

	sock := m.socketPath("a")
	if err := os.MkdirAll(filepath.Dir(sock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sock, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := m.execRun(context.Background(), "a"); err != nil {
		t.Fatalf("execRun: %v", err)
	}
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Fatalf("stale socket not removed: %v", err)
	}
}
