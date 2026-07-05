# syntax=docker/dockerfile:1

# Select how the `eg` CLI enters the image:
#   EG_SOURCE=git   (default) build eg from the okayeg repo at OKAYEG_REV
#   EG_SOURCE=local skip the in-image build; a prebuilt binary is bind-mounted
#                   over /usr/local/bin/eg at runtime (see docker-compose.yml's
#                   EG_BIN_HOST volume). Lets you iterate on okayeg by rebuilding
#                   in the okayeg tree with `cargo build --release -p eg` and
#                   pointing EG_BIN_HOST at the resulting binary (must be a
#                   musl/static build for the alpine base).
# BuildKit only builds the stage the final image references, so the unused path
# costs nothing.
ARG EG_SOURCE=git

# Build eg from the okayeg git remote, pinned to a rev for reproducibility.
# edition-2024 needs Rust 1.85+; 1.96 has been verified for this workspace.
FROM rust:1.96-alpine AS eg-git
RUN apk add --no-cache build-base
ARG OKAYEG_REV=1da42e41857d0baee7a3c1acf7822deaaf91f195
RUN cargo install eg --git https://github.com/HumanoidSandvichDispenser/okayeg.git --rev ${OKAYEG_REV} --root /out

# Placeholder stage for the local path: the real binary is bind-mounted at
# runtime, but the dev stage's COPY --from=eg still needs something to copy so
# /usr/local/bin/eg exists as a mount point.
FROM alpine:3 AS eg-local
RUN mkdir -p /out/bin && touch /out/bin/eg

FROM eg-${EG_SOURCE} AS eg

FROM golang:1.26-alpine AS dev
WORKDIR /app

# build-base: cgo toolchain for the sqlite driver
RUN apk add --no-cache build-base curl
RUN go install github.com/air-verse/air@latest

# the eg daemon hosts each project's share dir (see internal/okayeg/host.go).
# With EG_SOURCE=local this is overwritten by the EG_BIN_HOST bind mount.
COPY --from=eg /out/bin/eg /usr/local/bin/eg

COPY . /app

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]