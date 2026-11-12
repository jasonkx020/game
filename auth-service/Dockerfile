FROM golang:1.23-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/auth-api ./cmd/auth-api && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/auth-api /app/auth-api
COPY --from=build /out/migrate /app/migrate
COPY migrations /app/migrations
ENV AUTH_HTTP_PORT=8081
EXPOSE 8081
CMD ["/app/auth-api"]
