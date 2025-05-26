.PHONY: run-dev build test pprof-mem test-mem pprof-cpu test-cpu test-race

run-dev:
	@go run main.go $(ARGS)

build:
	@go build -o reallogs main.go

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
