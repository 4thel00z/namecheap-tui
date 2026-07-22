.PHONY: build test lint fmt hooks run clean

build:
	CGO_ENABLED=1 go build -o bin/ncp ./cmd/ncp

test:
	CGO_ENABLED=1 go test -race ./...

lint:
	golangci-lint run

fmt:
	gofumpt -l -w .

hooks:
	pre-commit install --install-hooks
	pre-commit install --hook-type commit-msg

run: build
	./bin/ncp

clean:
	rm -rf bin
