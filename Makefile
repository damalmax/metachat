format:
	gofmt -s -w .
	goimports -w .
	go mod tidy

test:
#	golangci-lint run ./...
	go test ./...

install: format test
	go install

docker:
	docker build -t thehadalone/metachat .