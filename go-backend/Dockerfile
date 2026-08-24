FROM golang:1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
# Bundled so migrations can be run in-place against the private DB network
# via `railway ssh`, without needing a public database connection string.
RUN CGO_ENABLED=0 GOOS=linux go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    mv /go/bin/migrate /out/migrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/api ./api
COPY --from=builder /out/migrate ./migrate
COPY migrations ./migrations
EXPOSE 8080
CMD ["./api"]
