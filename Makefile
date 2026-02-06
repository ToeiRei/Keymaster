.PHONY: fmt vet lint test check staticcheck vulncheck fix housekeeping build release install clean

# Versioning
VERSION ?= dev
GO_BUILD=go build

ifeq ($(OS),Windows_NT)
BIN_NAME=keymaster.exe
RELEASE_BIN=dist/keymaster.exe
else
BIN_NAME=keymaster
RELEASE_BIN=dist/keymaster
endif

fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	which golangci-lint >/dev/null 2>&1 || echo "golangci-lint not installed; run 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'"
	golangci-lint run

test:
	go test ./... -v -race


staticcheck:
	staticcheck ./...

vulncheck:
	govulncheck ./...

fix:
	go fix ./...

housekeeping: fmt fix vet staticcheck vulncheck test
	@echo "Housekeeping complete."

check: fmt vet lint test

# Build (debug/dev)
build:
	$(GO_BUILD) -o $(BIN_NAME) .

# Release (stripped, versioned)
release:
	@mkdir -p dist
	$(GO_BUILD) -ldflags "-s -w -X 'main.version=$(VERSION)'" -o $(RELEASE_BIN) .

# Install to $GOBIN, $GOPATH/bin, or /usr/local/bin
install: build
	@if [ -n "$$GOBIN" ]; then \
		cp $(BIN_NAME) "$$GOBIN/$(BIN_NAME)" && echo "Installed to $$GOBIN/$(BIN_NAME)"; \
	elif [ -n "$$GOPATH" ]; then \
		cp $(BIN_NAME) "$$GOPATH/bin/$(BIN_NAME)" && echo "Installed to $$GOPATH/bin/$(BIN_NAME)"; \
	else \
		cp $(BIN_NAME) /usr/local/bin/$(BIN_NAME) 2>/dev/null && echo "Installed to /usr/local/bin/$(BIN_NAME)" || echo "Copy to /usr/local/bin failed (try sudo)"; \
	fi

clean:
	rm -f $(BIN_NAME)
	rm -rf dist
