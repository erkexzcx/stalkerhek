#!/usr/bin/env bash

# Remove old binaries (if any)
rm -rf dist

# Build Linux binaries:
env GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o "dist/stalkerhek_linux_i386" ./cmd/stalkerhek/main.go             # Linux i386
env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "dist/stalkerhek_linux_x86_64" ./cmd/stalkerhek/main.go         # Linux 64bit
env GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="-s -w" -o "dist/stalkerhek_linux_arm" ./cmd/stalkerhek/main.go      # Linux armv5/armel/arm (it also works on armv6)
env GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o "dist/stalkerhek_linux_armhf" ./cmd/stalkerhek/main.go    # Linux armv7/armhf
env GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "dist/stalkerhek_linux_aarch64" ./cmd/stalkerhek/main.go        # Linux armv8/aarch64

# Build FreeBSD binary:
env GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w" -o "dist/stalkerhek_freebsd_x86_64" ./cmd/stalkerhek/main.go     # FreeBSD 64bit

# Build MacOS binary:
env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "dist/stalkerhek_darwin_x86_64" ./cmd/stalkerhek/main.go       # Darwin 64bit

# Build Windows binaries:
env GOOS=windows GOARCH=386 go build -ldflags="-s -w" -o "dist/stalkerhek_windows_i386.exe" ./cmd/stalkerhek/main.go     # Windows 32bit
env GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "dist/stalkerhek_windows_x86_64.exe" ./cmd/stalkerhek/main.go # Windows 64bit

# Compress binaries (risk of binary not working at all on some platforms):
# upx --best dist/*
