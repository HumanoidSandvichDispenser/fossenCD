FROM golang:1.26-alpine AS dev

WORKDIR /app

# build-base: cgo toolchain for the sqlite driver
RUN apk add --no-cache build-base curl
RUN go install github.com/air-verse/air@latest

# teamtype daemon hosts each project's share dir (see internal/teamtype/host.go)
ARG TEAMTYPE_VERSION=v0.9.1
ARG TEAMTYPE_ARCH
RUN arch="${TEAMTYPE_ARCH:-$(uname -m)}"; \
	case "$arch" in \
		x86_64 | aarch64) ;; \
		*) echo "unsupported arch: $arch" >&2; exit 1 ;; \
	esac; \
	curl -sL "https://github.com/teamtype/teamtype/releases/download/${TEAMTYPE_VERSION}/teamtype-${arch}-unknown-linux-musl.tar.gz" \
	| tar xz -C /usr/local/bin teamtype

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
