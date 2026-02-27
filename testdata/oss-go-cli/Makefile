.PHONY: build test lint clean

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/gosync .

test:
	go test -race -cover ./...

lint:
	go vet ./...
	golangci-lint run

clean:
	rm -rf bin/

install: build
	cp bin/gosync $(GOPATH)/bin/