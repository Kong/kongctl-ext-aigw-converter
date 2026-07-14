BINARY := kongctl-ext-ai-gateway-converter

.PHONY: build clean test

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -o bin/$(BINARY) ./cmd/$(BINARY)

test:
	go test ./...

clean:
	rm -rf bin dist
