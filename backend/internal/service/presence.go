package service

import (
	"sort"
	"sync"
	"time"
)

// Peer is one live peer on a project, as reported by the eg daemon's session
// hook. Name and Email are the peer's claim, shown verbatim; Verified and
// Username come from checking that claim against the endpoint's owning
// accounts.
type Peer struct {
	EndpointID string

	Name string

	Email string

	Verified bool

	// Username of the account the claimed email verified against.
	Username string

	JoinedAt time.Time

	// sessions is how many live connections this endpoint holds; the peer
	// drops from the list when it reaches zero.
	sessions int
}

// Presence tracks each project's live peers in memory, fed by the session hook
// events (hello at join, bye at leave). It restarts empty: entries reappear as
// running daemons' peers reconnect.
type Presence struct {
	mu       sync.Mutex
	projects map[string]map[string]*Peer
}

func NewPresence() *Presence {
	return &Presence{projects: make(map[string]map[string]*Peer)}
}

// Hello records a peer joining projectID, replacing the claim of an endpoint
// already present.
func (p *Presence) Hello(projectID string, peer Peer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peers := p.projects[projectID]
	if peers == nil {
		peers = make(map[string]*Peer)
		p.projects[projectID] = peers
	}

	if existing := peers[peer.EndpointID]; existing != nil {
		peer.sessions = existing.sessions
		peer.JoinedAt = existing.JoinedAt
	} else {
		peer.JoinedAt = time.Now()
	}
	peer.sessions++
	peers[peer.EndpointID] = &peer
}

// Bye records one of endpointID's sessions on projectID ending; the peer stays
// listed while it holds others.
func (p *Presence) Bye(projectID, endpointID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	peers := p.projects[projectID]
	peer := peers[endpointID]
	if peer == nil {
		return
	}

	peer.sessions--
	if peer.sessions <= 0 {
		delete(peers, endpointID)
		if len(peers) == 0 {
			delete(p.projects, projectID)
		}
	}
}

// Peers returns projectID's live peers in join order.
func (p *Presence) Peers(projectID string) []Peer {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]Peer, 0, len(p.projects[projectID]))
	for _, peer := range p.projects[projectID] {
		out = append(out, *peer)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out
}
