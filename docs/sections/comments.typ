= Comments and Suggestions <comments>

Comments and suggestions must reach every collaborator, including local
Teamtype replicas that never contact the backend. They must live inside the
synchronized document. The model below is similar to how git commits can be
signed, but each entity has its own way of verifying and trusting parties.

== Persistence

Comments are stored as files in a synchronized path in the working tree (e.g.
`.fossencd/`). These files would ride the CRDT, so that every peer (including
local Teamtype users) receives them for free, offline, with no separate
transport. The cost is that local users see comment files in their tree, but
this is acceptable and arguably git-like.

== Immutability

A signed object cannot be edited in place without breaking its signature, so
comments are _immutable_. This is turned into an advantage: the comment store is
an append-only set of immutable signed objects, exactly like git objects.

- Editing a comment is a new signed object that supersedes the prior one.
- Resolving or threading is new objects referencing parents.
- The CRDT therefore only ever _grows_. No two peers concurrently mutate the
  same comment blob, so the conflict problem of mutable comment files
  disappears entirely.

Each object's signature covers a canonical serialization of: the author
identity, the content, the anchor, the file path, a timestamp, any parent
reference, and an optional staleness hash (the hash of the text contained in
the comment's selected range). Any tampering or post-hoc edit breaks the
signature.

=== Preventing Tampering

The signature is a non-repudiation proof of authorship but does not prevent
deletion or replacement at the filesystem layer by a malicious party with write
access to the repository. For most use cases, a shared document implies some
sort of mutual trust, so this is not a problem.

But to solve this problem, because automerge is inherently append-only, any
comment object that is deleted or replaced by a malicious party would still
exist in the CRDT history. The caveat is that recovery is a manual search
through the history, around a $O(n)$ time complexity for $n$ CRDT operations.

A question for the future is how to surface this in the UI? One possibility is
showing a warning that "this exists in the backup but is missing from your
view; restore?" Without that UX, the append-only guarantee serves as only a
safety net that nobody looks at, but it is still a strong guarantee that
comment data is not truly lost.

== Anchoring

A comment points at a range in a document, but every keystroke shifts byte
offsets. Anchors therefore do not store offsets; they store an Automerge
cursor (a stable, edit-tracking reference) into the target file's text. Because
teamtype mirrors the whole directory as a single Automerge document, a comment
object in `.fossencd/` can reference a cursor in `main.typ` and follow its text
through edits.

The signed payload also includes a hash of the anchored substring at creation
time. When the underlying text later changes, the live text no longer matches
the hash, and the comment can be rendered _outdated_ --- the equivalent of
GitHub graying out a review comment whose lines have moved.

== Suggestions

A suggestion extends a comment: it additionally carries a proposed replacement
for its anchored range and an accept/reject state. Accepting a suggestion
applies the edit to the document. It reuses the same anchoring, signing, and
immutability machinery, plus the patch payload and resolution state.

== Signing

Comment authorship rides the signature, decoupling it from the throwaway
transport and CRDT identities. There are two signing paths, mirroring git and
GitHub:

- *Local signing:* A local Teamtype user signs the object with their own
  durable signing key, similarly to `git commit -S`. This works fully offline
  without any server involved. The friction of holding and managing a key is
  accepted, by design. Note this is a _user-level_ signing key, not the
  per-directory `.teamtype/key` transport key.
- *Server-assisted signing:* A web user can not use or upload a private key to
  use for signing, so they contact the fossenCD service to sign off for them,
  similarly to GitHub's `web-flow` user signing off the user's web commits.

== Verification

Verification is local-first, like git, which never auto-verifies:

- *Local verification:* A local client verifies signatures itself against a
  keyring assembled from the keyservers it is configured to trust.
- *Server-assisted verification:* The backend can auto-verify and surface a
  badge in the web UI.

From fossenCD's point of view, a comment is considered verified if _at least
one_ public key, from _any_ of the configured keyservers, validates its
signature (OR across the trusted keyserver union). The set of keyservers is
configured by the instance administrator. This is useful if an organization
might run its own keyserver for its members, or if a public instance
administrator chooses to trust a broad public keyserver plus its own.

=== Keyserver Federation

The instance itself is a keyserver, similar to GitHub: users upload their own
public keys, and the server serves them for verification. Two federation
features extend this:

- The instance administrator can add other keyservers as lookup sources, so the
  instance can resolve and trust keys it does not itself hold.
- Users can upload their own public keys to their account.
