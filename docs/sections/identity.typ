= Identity and trust <identity>

fossenCD juggles three distinct identities. They are kept separate and bound
together only where a binding is actually meaningful.

== Identity Layers

=== Managed by Teamtype

- *iroh NodeId (transport):* An ed25519 public key. It is the cryptographic
  identity of a peer's network endpoint. Connections are authenticated against
  it by QUIC/TLS, and access to a project is additionally gated by a passphrase
  exchanged in the handshake. The secret address a peer connects by is
  `<host-node-id>#<passphrase>`.
- *Automerge ActorId (authorship):* A per-peer identifier Automerge uses to
  attribute each change in the CRDT. It is what distinguishes "who made this
  edit" at the document level, independent of the transport. A single peer's
  cursors are keyed as `<actor-id>-<editor-id>`.

=== Managed by fossenCD

- *fossenCD account:* A durable user in the backend's database. The account
  identity owns projects, holds membership, and public keys. This is the only
  layer that corresponds to a _person_ across sessions and projects.

== Key Material

A Teamtype instance stores the private key in `.teamtype/key`, which the public
key can be derived from. The public key is not stored anywhere but is used for
registering with iroh relays and identifying iroh nodes.

fossenCD's backend reads and writes that same key file, so it can derive a
project's secret address (and rotate it by changing the file) without the
daemon running. Every local Teamtype peer, whether it ran `share` or `join`,
has its own `.teamtype/key`, hence its own NodeId.

=== Web Peer

The browser peer is different: it has no filesystem to persist a key, and it
does not store one. Currently, it generates a fresh keypair in memory per
session, although this optionally could be stored in `localStorage` in the
future. This means:

- Every page load is a brand-new peer.
- Therefore authorship that must survive a session (such as comments and
  suggestions) cannot rely on the Automerge ActorId or iroh NodeId. It must be
  anchored to the fossenCD account, via a signature (see @comments).

== Binding the Layers

The only binding that carries weight is _account to signing key_. A comment's
authorship is established by a signature whose key the backend (acting as a
keyserver) associates with a fossenCD account.
