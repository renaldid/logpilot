BINARY     = logpilot
MODULE     = github.com/renaldid/logpilot
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    = -ldflags "-X $(MODULE)/cmd.Version=$(VERSION)"

.PHONY: build run test test-coverage lint clean release-dry release demo

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

run:
	go run $(LDFLAGS) .

test:
	go test ./...

test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ coverage.out coverage.html

release-dry:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean

demo:
	vhs docs/demo.tape
