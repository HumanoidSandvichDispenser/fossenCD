# fossenCD

fossenCD (free and open-source software, editor-native collaborative documents)
is a self-hostable collaborative document editor built on top of Typst, a
modern typesetting system. It uses
[teamtype](https://github.com/teamtype/teamtype), a P2P collaborative editing
protocol and system and communicates through a [WebAssembly peer
implementation](https://github.com/humanoidsandvichdispenser/teamtype).

This allows users to both edit documents collaboratively using a web-based
editor and also edit documents locally with their own text editors, all while
maintaining real-time synchronization across all peers. This combines the
approachability of web-based text editing for users who prefer it with the
flexibility of local text editing for users who prefer that workflow.

The [design document](docs/main.pdf) covers the full architecture.

## WHAT IS THIS PROJECT ABOUT????

- Edit with multiple partners on the same Doc
- Need to invite Team Pepeja to the project? No problem, just share the project
  link and they can join in real-time!
- Export your Docs to PDF when needed
- Licensed under AGPL-3.0, with complete transparency for users of the entire
  protocol, client, and frontend back-to-back
- Users can also edit documents locally with other text editors like Neovim and
  Emacs and third party things
- Supports Typst incremental compilation, so even minor mistakes won't cause
  the entire document to recompile
- Peer client written in Rust, so not only is it blazing fast, it also has
  SPEED, VIOLENCE, MOMENTUM with every edit
- Put em around the La Casa with the ability to self-host the entire system,
  stuff like that

## Frequently Asked Questions

### What does the CD stand for?

Something about Collaborative Documents being transparent and free and open
source, etc etc. It's like, y'know, these little chubby cheek wannabes editing
documents together.

### What the hell is pepeja?

[Pepeja definition](https://imgur.com/RxbVYe7)
