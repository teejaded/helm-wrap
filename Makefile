.PHONY: test build deps
BINARY=helm-wrap

default: build

test:
	go vet github.com/teejaded/helm-wrap
	go test -v -coverprofile cover.out ./...

build:
	goreleaser build --clean --snapshot

deps:
	go install github.com/goreleaser/goreleaser@latest
