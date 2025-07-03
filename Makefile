.PHONY: run-dev build test pprof-mem test-mem pprof-cpu test-cpu test-race

VERSION ?= dev
TAG ?= v1.1.0

run-dev:
	@go run main.go $(ARGS)

build:
	@go build -ldflags "-X main.Version=$(VERSION)" -o reallogs main.go

build-tagged:
	@$(MAKE) build VERSION="$(TAG)"

add-tag:
	@git tag -a $(TAG) -m "Release de la versión $(TAG)"

test:
	@go test -v ./...

pprof-mem:
	@go tool pprof memprofile.prof

pprof-cpu:
	@go tool pprof cpuprofile.prof

test-mem:
	@go build -o reallogs main.go && ./reallogs -flow=fromdir -dir=logs-f -memprofile=memprofile.prof

test-cpu:
	@go build -o reallogs main.go && ./reallogs -flow=fromdir -dir=logs-f -cpuprofile=cpuprofile.prof

test-race:
	@go run -race main.go -flow=fromdir -dir=logs-f
