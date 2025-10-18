# syntax=docker/dockerfile:1
# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o igpu-exporter ./cmd

# Runtime stage
FROM ubuntu:22.04

# Install intel-gpu-tools
RUN apt-get update
RUN apt-get install -y intel-gpu-tools && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/igpu-exporter .

ENV IS_DOCKER=1

# Expose Prometheus metrics port
EXPOSE 8080

# Run the exporter
ENTRYPOINT ["/app/igpu-exporter"]
