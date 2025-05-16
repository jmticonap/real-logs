.PHONY: run-dev build

run-dev:
	@go run main.go $(ARGS)

build:
	@go build -o reallogs main.go