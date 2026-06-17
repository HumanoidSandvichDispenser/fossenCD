= Overview

fossenCD (Free and Open-Source Software, Editor-Native Collaborative Documents)
is a self-hostable collaborative document editor for Typst. It is built on
#link("https://github.com/teamtype/teamtype")[teamtype], a peer-to-peer
collaborative editing protocol (with an included Rust implementation) that
synchronizes a shared Automerge CRDT document over iroh (for relaying) and
Magic Wormhole (for out-of-band join codes). The defining property of teamtype
is that it is fully peer-to-peer and local-first: the document is stored in a
local directory, and a teamtype daemon synchronizes it with other peers over
the network, but there is no required central server or service, nor do peers
have to be online at the same time.

fossenCD builds on this with a Go backend that provides project hosting with
persistent teamtype peers (for users to connect to), a Vue frontend for web
editing, and an identity and access control layer, which altogether allows both
local Teamtype peers and web peers to collaborate on the same document.

In other words, fossenCD is to teamtype roughly what Gitea is to git. Just as
Gitea provides a self-hostable higher-level identity layer over git's
distributed version control, fossenCD provides accounts, access control, a web
editing experience, and an always-online peer on top of teamtype's peer-to-peer
CRDT synchronization protocol, scoped deliberately to Typst the way Overleaf
scopes itself to LaTeX.

== Editing modes

fossenCD supports two first-class ways to edit the same document, concurrently:

- *Local:* The user runs the teamtype daemon against a directory on their own
  machine and edits the files with any editor (e.g. Neovim, Emacs, VS Code).
  They are a durable peer with a persistent identity and full offline history.
- *Web:* The user opens the project in a browser. A WebAssembly teamtype peer
  runs in the page, connects to the project's host, and synchronizes live. This
  peer is ephemeral and holds no persistent identity and no local history
  beyond the session.

Both modes speak the same protocol and edit the same CRDT, so a local user in
Neovim and a web user in the browser collaborate on the same document in real
time.

== Goals

- Self-hostable end to end; no required dependency on any cloud service we run.
- Local and web peers as equal collaborators on one document.
- Real-time collaboration with offline-capable local replicas.
- Identity and access control that fossenCD owns, layered cleanly over
  teamtype's transport and CRDT identities.
- Signed, verifiable comments and suggestions that work locally without
  contacting a server.
- The protocol, peer client, and frontend are all open source (AGPL-3.0).

== Non-goals

- Not a general-purpose document platform. Currently, this project only aims to
  support Typst.
- Not a centralized SaaS.
- Not a replacement for teamtype; fossenCD depends on it and tracks upstream.

== Design principles

/ The synced document is the only thing every peer sees: Local Teamtype
  replicas speak only teamtype's P2P protocol, and so the fossenCD service
  appears invisible. Anything _every_ collaborator must observe therefore has
  to live inside the synchronized CRDT.

/ Identity is layered: Transport identity (iroh via teamtype), authorship
  identity (Automerge via teamtype), and account identity (fossenCD) are
  distinct, and bound to each other explicitly where needed rather than assumed
  equal.

/ The system provides convenience and is not a single point of truth: An
  example is that secret addresses and join codes needed for peers to connect
  can be derived deterministically (by deriving public key from `.teamtype/key`
  for iroh nodes and generating a passphrase required by each peer, which is
  exchanged through Magic Wormhole). In fossenCD, the system prefers to not ask
  the daemon and instead derives the values by itself, but the core derivation
  must remain the same as how the teamtype daemon would derive them, and the
  backend should _not_ own the keys.

/ Follow git's trust model: Local-first signing and verification, with the
  server as an auto-verifier (plus additional keyservers).

== Tech stack

- *Synchronization:* Teamtype (Automerge CRDT, iroh transport, magic-wormhole
  for join codes)
- *Web peer:* teamtype-wasm, a WebAssembly build of a Teamtype peer
  (`@sandvichxyz/teamtype-wasm`)
- *Backend:* Go: chi + huma for the HTTP/OpenAPI layer, GORM over SQLite for
  persistence
- *Frontend:* Vue 3 + TypeScript + Pinia, with a generated OpenAPI client
- *Reverse Proxy:* nginx
