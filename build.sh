#!/usr/bin/env bash
set -e

APP_NAME="dayswithout"
VERSION=$(date +%Y.%m.%d-%H%M)

echo "[INFO] Building $APP_NAME version $VERSION"

# Clean previous builds
rm -rf build
mkdir -p build

# Build for current OS/ARCH
echo "[INFO] Building for current system..."
go build -o build/$APP_NAME main.go

# Example: cross-compile for Linux amd64
echo "[INFO] Building for linux/amd64..."
GOOS=linux GOARCH=amd64 go build -o build/${APP_NAME}-linux-amd64 main.go

# Example: cross-compile for Linux arm64
echo "[INFO] Building for linux/arm64..."
GOOS=linux GOARCH=arm64 go build -o build/${APP_NAME}-linux-arm64 main.go

# Example: cross-compile for Windows
echo "[INFO] Building for windows/amd64..."
GOOS=windows GOARCH=amd64 go build -o build/${APP_NAME}-windows-amd64.exe main.go

# Example: cross-compile for macOS
echo "[INFO] Building for darwin/amd64..."
GOOS=darwin GOARCH=amd64 go build -o build/${APP_NAME}-darwin-amd64 main.go

echo "[INFO] Build finished. Files are in ./build/"
ls -lh build/
