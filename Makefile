run:
	go mod tidy && go build -o bin/golang-test-task cmd/main.go
test:
	go mod tidy && go test -v ./...