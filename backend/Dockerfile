FROM golang:1.26-alpine AS dev

WORKDIR /app

# build-base: cgo toolchain for the sqlite driver
RUN apk add --no-cache build-base
RUN go install github.com/air-verse/air@latest

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
