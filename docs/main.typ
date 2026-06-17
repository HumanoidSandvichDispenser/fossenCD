#import "@preview/grape-suite:4.0.0": exercise
#import exercise: project, task, subtask

#show: project.with(
  title: "fossenCD Design Document",
  abstract: [
    fossenCD is a self-hostable collaborative Typst editor built on teamtype, a
    peer-to-peer CRDT editing protocol over iroh. It bridges native peers
    (people editing files locally with their own editor + the teamtype daemon)
    and web peers (people editing in the browser via a WebAssembly peer), with
    a Go backend providing accounts, projects, membership, and supervised
    hosting, and a Vue frontend for the web editing experience. This document
    describes the architecture, the identity and trust model, project hosting,
    and the design for signed comments and suggestions.
  ],
)

#outline(title: "Contents", depth: 3, indent: 1em)

#pagebreak()

#include "sections/overview.typ"
#include "sections/identity.typ"
#include "sections/hosting.typ"
#include "sections/comments.typ"
