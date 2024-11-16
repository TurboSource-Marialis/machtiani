#!/bin/bash

# Function to display usage information
usage() {
  echo "Usage: $0 [--release]"
  exit 1
}

# Check for the --release flag
RELEASE=false

for arg in "$@"; do
  case $arg in
    --release)
      RELEASE=true
      shift # Remove --release from the arguments
      ;;
    *)
      usage
      ;;
  esac
done

if [ "$RELEASE" = true ]; then
  # Generate ldflags
  LD_FLAGS=$(./generate_ldflags)

  # Build for macOS (Intel)
  GOOS=darwin GOARCH=amd64 go build -ldflags "$LD_FLAGS" -o machtiani-darwin-amd64 ./cmd/machtiani

  # Build for macOS (Apple Silicon)
  GOOS=darwin GOARCH=arm64 go build -ldflags "$LD_FLAGS" -o machtiani-darwin-arm64 ./cmd/machtiani

  # Build for Linux (x86_64)
  GOOS=linux GOARCH=amd64 go build -ldflags "$LD_FLAGS" -o machtiani-linux-amd64 ./cmd/machtiani

else
  # Generate ldflags
  LD_FLAGS=$(./generate_ldflags)

  # Build the main application with ldflags
  go build -ldflags "$LD_FLAGS" -o machtiani ./cmd/machtiani
fi
