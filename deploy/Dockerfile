# Multi-stage build: compile on any host, run on Linux (amd64).
#   docker build -f deploy/Dockerfile --target platform-api -t game-platform-api .
#   docker build -f deploy/Dockerfile --target game -t game-server .

FROM golang:1.23-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/platform-api ./cmd/platform-api
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/game ./cmd/game

FROM alpine:3.20 AS platform-api
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/platform-api .
EXPOSE 8080
USER nobody
ENTRYPOINT ["./platform-api"]

FROM alpine:3.20 AS game
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/game .
EXPOSE 3250
USER nobody
ENTRYPOINT ["./game"]
