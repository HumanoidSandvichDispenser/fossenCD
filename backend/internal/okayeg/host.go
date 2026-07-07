package okayeg

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// HostOptions configures a HostManager.
type HostOptions struct {
	Bin            string // eg binary (on PATH or absolute); defaults to "eg"
	DataDir        string // base dir holding per-project share dirs
	RestartBackoff time.Duration
	StopGrace      time.Duration
}

// Host runs and supervises the okayeg `eg serve` daemon that hosts a project's
// document, so web peers can reach it at the project's endpoint id. Call
// EnsureHost when a peer needs the project (e.g. on endpoint fetch or peer
// registration); call Stop before deleting its share dir.
type Host interface {
	// EnsureHost starts a daemon hosting id's document if one isn't running.
	EnsureHost(id string)
	// Stop terminates id's daemon and waits for exit.
	Stop(id string)
	// Revoke closes endpointID's live sessions on id's daemon, via the repl on
	// the daemon's stdin. A no-op when no daemon is running (there is nothing
	// to close; the peer's next connection is the gate's call).
	Revoke(id, endpointID string) error
	// Logs returns the recent stdout+stderr captured for id's daemon, and
	// whether a host is currently registered for id.
	Logs(id string) (string, bool)
}

// maxLogBytes caps how much daemon output is retained per host.
const maxLogBytes = 64 << 10

var _ Host = (*HostManager)(nil)

// HostManager supervises one `eg serve` process per project, restarting it if
// it exits until Stop is called.
type HostManager struct {
	opts  HostOptions
	base  context.Context
	mu    sync.Mutex
	hosts map[string]*hostProc
	run   func(ctx context.Context, id string) error
}

type hostProc struct {
	cancel context.CancelFunc
	done   chan struct{}
	log    *logBuffer
	stdin  io.WriteCloser // the running daemon's repl, nil until it spawns
}

// logBuffer is a byte ring capturing the most recent daemon output.
type logBuffer struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = b.buf[len(b.buf)-b.max:]
	}
	return len(p), nil
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// NewHostManager builds a HostManager. The base context bounds every spawned
// daemon's lifetime: cancelling it stops all hosts.
func NewHostManager(base context.Context, opts HostOptions) *HostManager {
	if opts.Bin == "" {
		opts.Bin = "eg"
	}
	if opts.RestartBackoff == 0 {
		opts.RestartBackoff = 2 * time.Second
	}
	if opts.StopGrace == 0 {
		opts.StopGrace = 5 * time.Second
	}
	m := &HostManager{
		opts:  opts,
		base:  base,
		hosts: make(map[string]*hostProc),
	}
	m.run = m.execRun
	return m
}

// EnsureHost ensures a host daemon is running for the given id. If not, it
// starts one.
func (m *HostManager) EnsureHost(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.hosts[id]; ok {
		return
	}
	ctx, cancel := context.WithCancel(m.base)
	h := &hostProc{cancel: cancel, done: make(chan struct{}), log: &logBuffer{max: maxLogBytes}}
	m.hosts[id] = h
	go m.supervise(ctx, id, h)
}

// Stop stops the host daemon for the given id by sending SIGTERM, then waits
// for it to exit.
func (m *HostManager) Stop(id string) {
	m.mu.Lock()
	h, ok := m.hosts[id]
	if ok {
		delete(m.hosts, id)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	h.cancel()
	<-h.done
}

// Revoke writes `revoke <endpointID>` to id's daemon over its stdin repl,
// closing the peer's live sessions. Whether the peer gets back in on its next
// connection is the authz gate's call, so callers change policy (the DB) first
// and then revoke.
func (m *HostManager) Revoke(id, endpointID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.hosts[id]
	if !ok || h.stdin == nil {
		return nil // no daemon, no live session to close
	}
	_, err := fmt.Fprintf(h.stdin, "revoke %s\n", endpointID)
	return err
}

// Logs returns the recent stdout+stderr captured for id's daemon, and whether a
// host is currently registered for id.
func (m *HostManager) Logs(id string) (string, bool) {
	m.mu.Lock()
	h, ok := m.hosts[id]
	m.mu.Unlock()
	if !ok {
		return "", false
	}
	return h.log.String(), true
}

func (m *HostManager) supervise(ctx context.Context, id string, h *hostProc) {
	defer close(h.done)
	for {
		// run the daemon
		_ = m.run(ctx, id)

		// restart if it exits, unless context is done
		if ctx.Err() != nil {
			return
		}
		// also bail if Stop already removed us
		m.mu.Lock()
		current := m.hosts[id] == h
		m.mu.Unlock()
		if !current {
			return
		}
		select {
		case <-time.After(m.opts.RestartBackoff):
		case <-ctx.Done():
			return
		}
	}
}

func (m *HostManager) execRun(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, m.opts.Bin, m.args(id)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("okayeg: host %s stdin: %w", id, err)
	}
	m.mu.Lock()
	h := m.hosts[id]
	if h != nil {
		h.stdin = stdin
		cmd.Stdout = h.log
		cmd.Stderr = h.log
	}
	m.mu.Unlock()
	// the daemon tears its network down on a signal, so stop gracefully with
	// SIGTERM; SIGKILL after StopGrace if it hangs. The stdin repl only lists
	// and revokes sessions, so there's nothing to clean up over it first.
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = m.opts.StopGrace
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("okayeg: start host %s: %w", id, err)
	}
	err = cmd.Wait()
	m.mu.Lock()
	if h != nil && h.stdin == stdin {
		h.stdin = nil
	}
	m.mu.Unlock()
	stdin.Close()
	return err
}

// args builds the `eg serve` argv. -C points eg at the project's share dir,
// which EnsureKey already seeded with .eg/key; `serve` self-initializes an
// empty doc when the dir has none.
func (m *HostManager) args(id string) []string {
	return []string{"-C", m.shareDir(id), "serve"}
}

func (m *HostManager) shareDir(id string) string {
	return filepath.Join(m.opts.DataDir, id)
}

// Args exposes the `eg serve` argv for tests so they don't have to run the daemon.
func (m *HostManager) Args(id string) []string { return m.args(id) }
