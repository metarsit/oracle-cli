.PHONY: build test lint cover fmt vet vuln release clean

BIN     := oracle
PKG     := ./cmd/oracle
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

test:
	go test ./...

cover:
	go test -coverprofile=cover.out ./...
	go tool cover -func cover.out | tail -1

fmt:
	gofmt -l -w .

vet:
	go vet ./...

lint:
	golangci-lint run

vuln:
	govulncheck ./...

clean:
	rm -rf bin cover.out dist

release:
	mkdir -p dist
	for os_arch in darwin/arm64 darwin/amd64 linux/amd64; do \
		os=$${os_arch%/*}; arch=$${os_arch#*/}; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
			-o dist/oracle-cli_$(VERSION)_$${os}_$${arch} $(PKG); \
	done
	cd dist && shasum -a 256 oracle-cli_* > SHA256SUMS
