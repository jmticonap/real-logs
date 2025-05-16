.PHONY: run-dev build

run-dev:
	@go run main.go

build:
	@go build -o reallogs main.go