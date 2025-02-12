.PHONY: ci build-release generate fmt test tidy lint tools

ci: tools test lint generate fmt tidy

.PHONY: generate
generate:
	go generate ./...

.PHONY: fmt
fmt:
	gosimports -l -w ./

.PHONY: test
test:
	go test -race ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run ./...

.PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.2
	go install github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8

.PHONY: build-release  # Separate build-release target
build-release:
	mkdir -p build/scuttlego  # Create the nested build directory

        # Example: Build for Linux (adjust as needed)
        GOOS=linux GOARCH=amd64 go build -o build/scuttlego/scuttlego-linux-amd64 ./...

        # Example: Build for macOS (adjust as needed)
        GOOS=darwin GOARCH=amd64 go build -o build/scuttlego/scuttlego-darwin-amd64 ./...

        # Example: Create zip archives (optional, but recommended)
        zip -r build/scuttlego/artifact1.zip build/scuttlego/scuttlego-linux-amd64
        zip -r build/scuttlego/artifact2.zip build/scuttlego/scuttlego-darwin-amd64

        # Add other platforms/architectures/archives as needed.
        # Ensure that the files created here match the paths in your GitHub Actions workflow.
