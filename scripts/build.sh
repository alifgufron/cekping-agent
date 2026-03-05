#!/bin/bash

# Build for various architectures
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/cekping-agent-linux-amd64 ./cmd/agent/main.go

echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -o bin/cekping-agent-linux-arm64 ./cmd/agent/main.go

echo "Building for FreeBSD (amd64)..."
GOOS=freebsd GOARCH=amd64 go build -o bin/cekping-agent-freebsd-amd64 ./cmd/agent/main.go

echo "Building for FreeBSD (arm64)..."
GOOS=freebsd GOARCH=arm64 go build -o bin/cekping-agent-freebsd-arm64 ./cmd/agent/main.go

echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o bin/cekping-agent-darwin-amd64 ./cmd/agent/main.go

echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o bin/cekping-agent-darwin-arm64 ./cmd/agent/main.go

echo "Done!"
