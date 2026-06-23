# Multi-stage build for Zero-Trust Swarm
FROM golang:alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy

# Build all binaries statically
ENV CGO_ENABLED=0
RUN go build -o /bin/idp-daemon ./cmd/idp-daemon
RUN go build -o /bin/registry ./cmd/registry
RUN go build -o /bin/agent ./cmd/agent
RUN go build -o /bin/sidecar ./cmd/sidecar

# Final Stage
FROM alpine:latest
WORKDIR /app

COPY --from=builder /bin/idp-daemon /app/idp-daemon
COPY --from=builder /bin/registry /app/registry
COPY --from=builder /bin/agent /app/agent
COPY --from=builder /bin/sidecar /app/sidecar

# Make binaries executable
RUN chmod +x /app/idp-daemon /app/registry /app/agent /app/sidecar
