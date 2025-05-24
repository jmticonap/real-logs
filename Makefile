.PHONY: run-dev build test

run-dev:
	@go run main.go $(ARGS)

build:
	@go build -o reallogs main.go

test:
	@go test -v ./...