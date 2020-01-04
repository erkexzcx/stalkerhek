#!/usr/bin/env bash

NAME="stalkerhek"

# Remove old binaries (if any)
rm -rf dist

# Build Linux binaries:
env GOOS=linux GOARCH=386 go build -ldflags="-s -w" -o "dist/${NAME}_linux_i386"             # Linux i386
env GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "dist/${NAME}_linux_x86_64"         # Linux 64bit
env GOOS=linux GOARCH=arm GOARM=5 go build -ldflags="-s -w" -o "dist/${NAME}_linux_arm"      # Linux armv5/armel/arm (it also works on armv6)
env GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o "dist/${NAME}_linux_armhf"    # Linux armv7/armhf
env GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "dist/${NAME}_linux_aarch64"        # Linux armv8/aarch64

# Build FreeBSD binary:
env GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w" -o "dist/${NAME}_freebsd_x86_64"     # FreeBSD 64bit

# Build MacOS binary:
env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "dist/${NAME}_darwin_x86_64"       # Darwin 64bit

# Build Windows binaries:
env GOOS=windows GOARCH=386 go build -ldflags="-s -w" -o "dist/${NAME}_windows_i386.exe"     # Windows 32bit
env GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "dist/${NAME}_windows_x86_64.exe" # Windows 64bit

# Compress binaries (risk of binary not working at all on some platforms):
# upx --best dist/*
