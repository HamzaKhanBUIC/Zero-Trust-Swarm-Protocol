# Multi-stage build for Zero-Trust Swarm
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ (Uncomment if using dependencies)
RUN go mod download

COPY . .

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
