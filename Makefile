build:
	dep ensure
	gofmt -s -w .
	goimports -w .
	golangci-lint run ./...
	go test ./...
	go install