# ============================
# 1️⃣ Build stage
# ============================
FROM golang:1.24-bullseye AS builder

WORKDIR /workspace

# Copy go mod files first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build statically for Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o widget-apiserver main.go

# ============================
# 2️⃣ Runtime stage
# ============================
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# Install minimal CA certificates for TLS verification
RUN microdnf install -y ca-certificates && microdnf clean all

# Copy binary from builder
COPY --from=builder /workspace/widget-apiserver /usr/local/bin/widget-apiserver

# Create nonroot user
RUN microdnf install -y shadow-utils && \
    useradd -u 65532 nonroot && \
    microdnf clean all

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/widget-apiserver"]
