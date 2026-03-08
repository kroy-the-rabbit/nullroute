APP := nullroute

.PHONY: build test run fmt

build:
	mkdir -p bin
	go build -o bin/$(APP) ./cmd/nullroute

test:
	go test ./...

run:
	go run ./cmd/nullroute -config ./examples/nullroute-config.yaml

fmt:
	gofmt -w ./cmd ./internal
