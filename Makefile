.PHONY: build test lint run docker-build clean

BINARY   := hooker
IMAGE    := hooker:latest
GO       := go
LDFLAGS  := -ldflags="-s -w"

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY) ./cmd/hooker

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...

run: build
	./bin/$(BINARY)

docker-build:
	docker build -t $(IMAGE) .

clean:
	rm -rf bin/
