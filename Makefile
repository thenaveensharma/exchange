# Build the Go application and create an executable in the bin directory
build:
	go build -o bin/exchange

# Run the application (automatically builds first)
run: build
	./bin/exchange

# Run all tests in the project with verbose output
test:
	go test -v ./...