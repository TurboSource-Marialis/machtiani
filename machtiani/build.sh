#!/bin/bash

# Generate ldflags
LD_FLAGS=$(./generate_ldflags)

# Build the main application with ldflags
go build -ldflags "$LD_FLAGS" -o machtiani ./cmd/machtiani

