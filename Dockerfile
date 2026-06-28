# ============================================================
#  Multi-stage Dockerfile for Go Load Balancer
#  Produces a minimal scratch image with all three binaries
# ============================================================

# ---- Stage 1: Build ----
FROM golang:1.26-alpine AS builder

# Install git + ca-certificates (needed for scratch)
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependencies first (go.mod has no external deps, but good practice)
COPY go.mod ./
RUN go mod download

# Copy the entire source tree
COPY . .

# Build all three binaries as static executables
# CGO_ENABLED=0 ensures fully static linking (no glibc dependency)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/loadbalancer ./cmd/loadbalancer
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/backend     ./cmd/backend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /bin/stresstest   ./cmd/stresstest

# ---- Stage 2: Runtime ----
FROM scratch

# Import CA certs from builder (needed for HTTPS if ever added)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the compiled binaries
COPY --from=builder /bin/loadbalancer /loadbalancer
COPY --from=builder /bin/backend     /backend
COPY --from=builder /bin/stresstest  /stresstest

# Expose the load balancer port
EXPOSE 8080

# Default entrypoint is the load balancer
ENTRYPOINT ["/loadbalancer"]
