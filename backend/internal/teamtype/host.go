package teamtype

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type HostOptions struct {
	Bin            string
	DataDir        string
	WormholeRelay  string
	IrohRelay      string
	IrohPkarr      string
	IrohDNSDomain  string
	RestartBackoff time.Duration
	StopGrace      time.Duration
}

// Host runs and supervises the teamtype peer daemon that hosts a project's
// document, so web peers can reach it at the project's secret address. Call
// EnsureHost when a peer needs the project (e.g. on address fetch); call Stop
// before deleting its share dir.
type Host interface {
	// EnsureHost starts a daemon hosting id's document if one isn't running.
	EnsureHost(id string)
	// Stop terminates id's daemon and waits for exit.
	Stop(id string)
}

var _ Host = (*HostManager)(nil)

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
}

func NewHostManager(base context.Context, opts HostOptions) *HostManager {
	if opts.Bin == "" {
		opts.Bin = "teamtype"
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
	h := &hostProc{cancel: cancel, done: make(chan struct{})}
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
	// the daemon only tears its network down on a signal, so stop gracefully
	// with SIGTERM; SIGKILL after StopGrace if it hangs
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = m.opts.StopGrace
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("teamtype: start host %s: %w", id, err)
	}
	return cmd.Wait()
}

// args builds the share argv. No --init (that resets history); share self-inits
// a doc when the share dir has none.
func (m *HostManager) args(id string) []string {
	a := []string{"--directory", m.shareDir(id), "share", "--no-join-code"}
	for flag, val := range map[string]string{
		"--magic-wormhole-relay": m.opts.WormholeRelay,
		"--iroh-relay":           m.opts.IrohRelay,
		"--iroh-pkarr-relay":     m.opts.IrohPkarr,
		"--iroh-dns-domain":      m.opts.IrohDNSDomain,
	} {
		if val != "" {
			a = append(a, flag, val)
		}
	}
	return a
}

func (m *HostManager) shareDir(id string) string {
	return filepath.Join(m.opts.DataDir, id)
}
