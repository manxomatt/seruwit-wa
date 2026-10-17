.PHONY: build build-linux run clean

# Build native binary for local OS
build:
	go build -o bin/seruwit-wa ./cmd/server

# Cross-compile standalone binary for Linux 64-bit (Production)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/seruwit-wa-linux-amd64 ./cmd/server

# Cross-compile for Linux ARM64 (AWS Graviton, etc.)
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/seruwit-wa-linux-arm64 ./cmd/server

# Run locally
run:
	go run ./cmd/server

clean:
	rm -rf bin/
