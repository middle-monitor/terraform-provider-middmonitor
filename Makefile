.PHONY: build test fmt docs

build:
	go build -o terraform-provider-middmonitor .

test:
	go test ./...

fmt:
	gofmt -s -w .

# Registry documentation, generated from the schemas and examples/.
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.20.1 generate --provider-name middmonitor
