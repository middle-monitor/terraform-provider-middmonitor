.PHONY: build test fmt

build:
	go build -o terraform-provider-middmonitor .

test:
	go test ./...

fmt:
	gofmt -s -w .
