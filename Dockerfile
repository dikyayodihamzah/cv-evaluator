## Multi-stage build for CV Evaluator
## Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build tools
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the binary (static)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags "-s -w" -trimpath -o /app/bin/cv-evaluator .

## Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && update-ca-certificates \
  && adduser -D -H -s /sbin/nologin app

WORKDIR /app

# Copy compiled binary
COPY --from=builder /app/bin/cv-evaluator /app/cv-evaluator

USER app

ENTRYPOINT ["/app/cv-evaluator"]