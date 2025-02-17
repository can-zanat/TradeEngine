build:
	go build cmd/main.go

run:
	go run cmd/main.go

lint:
	golangci-lint run -c .golangci.yml

unit-test:
	go test ./... -short

generate-mocks:
	mockgen -source=./internal/store.go -destination=./internal/mock_store.go -package=internal
