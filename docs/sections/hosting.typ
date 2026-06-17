= Project Hosting

For any Teamtype peer to join a share (in particular, for a fossenCD web user
to open a project), _some_ peer must be online to synchronize with. A hosted
fossenCD instance fills that role by running an instance of teamtype per
project to accept connections and synchronize changes at any time.

== Addresses and join codes

The backend derives a project's secret address directly from its
`.teamtype/key` (node id from the seed, passphrase from the key's second half),
so addresses can be served over the API without involving the daemon. A join
code is a fresh random magic-wormhole code. The backend mints it in-process and
hands the key-derived secret address over the wormhole channel when a peer
claims it, rather than scraping the code from daemon output.

== Access control and revocation <access-control>

Project access is gated by a passphrase. In vanilla Teamtype, that passphrase
_is_ the second half of `.teamtype/key` (see @identity). So the secret address
`nodeid#passphrase` is one capability shared by everyone, and the only way to
revoke a member is to rotate the key (generate a new `.teamtype/key`), which
yields a new NodeId, address, and passphrase, then redistribute the new address
to the members who remain on request. This breaks for local users who join
through `teamtype join` with a configured secret address without passing in a
join code. There are two ways to be able to solve this:

+ Fork `teamtype` to make a persistent server version, and give each peer
  a unique passphrase (which must be stored, ideally in `.teamtype` to persist
  across restarts)
+ Expose a convenient way (like a `curl | sh` script) to retrieve the current
  secret address; currently we expose an authenticated endpoint for retrieving
  the secret address, and this just takes the response and writes it to the
  Teamtype config

Option 2 is currently the easiest way to make the revocation UX acceptable. In
the future, a dedicated Teamtype server would be beneficial in the long run to
provide further granular access control and other features.
