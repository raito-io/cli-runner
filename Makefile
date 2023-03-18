# To try different version of Go
GO := go

build:
	go build .

test:
	go test ./...

lint:
	golangci-lint run ./...
	go fmt ./...

