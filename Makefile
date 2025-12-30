.PHONY: fmt vet lint test check

fmt:
	@gofmt -s -l . | sed -n '1,200p'

vet:
	@go vet ./...

lint:
	@which golangci-lint >/dev/null 2>&1 || echo "golangci-lint not installed; run 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'"
	@golangci-lint run

test:
	@go test ./... -v -race

check: fmt vet lint test
