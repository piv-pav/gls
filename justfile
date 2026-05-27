# Build gls binary
build:
    go build -ldflags "-X main.Version=$(cat .version)" -o gls .

# Install gls using go install
install:
    go install -ldflags "-X main.Version=$(cat .version)" .
    @echo "Installed to $(go env GOPATH)/bin/gls"

# Run tests
test:
    go test -v ./...

# Clean build artifacts
clean:
    rm -rf bin/ gls

# Install dependencies
deps:
    go mod download
    go mod tidy

# Format code
fmt:
    go fmt ./...

# Run linter
lint:
    go vet ./...

# Run example
run-example: build
    mkdir -p /tmp/example-docs
    echo "# Example Document\nThis is a test document about Go programming." > /tmp/example-docs/test.md
    ./gls index /tmp/example-docs
    ./gls search "programming"
    ./gls stats
