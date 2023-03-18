# To try different version of Go
GO := go

build:
	go build main.go

test:
	go test ./...

lint:
	golangci-lint run ./...
	go fmt ./...

