ci: tools test lint generate fmt tidy.PHONY: generate
generate:
	go generate./....PHONY: fmt
fmt:
	gosimports -l -w./.PHONY: test
test:
	go test -race./....PHONY: tidy
tidy:
	go mod tidy.PHONY: lint
lint:
	go vet./...
	golangci-lint run./....PHONY: tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.2
	go install github.com/rinchsan/gosimports/cmd/gosimports@v0.3.8.PHONY: build-release
build-release:
	rm -rf build
	mkdir -p build/scuttlego/cmd/log-debugger/linux-amd64
	GOOS=linux GOARCH=amd64 go build -v -o build/scuttlego/cmd/log-debugger/linux-amd64/log-debugger./cmd/log-debugger

	mkdir -p build/scuttlego/cmd/log-debugger/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -v -o build/scuttlego/cmd/log-debugger/darwin-amd64/log-debugger./cmd/log-debugger

        # Other platforms for log-debugger

        # Only include the following if scuttlego is a main package
        # with a main function you want to build:
	mkdir -p build/scuttlego/linux-amd64
	GOOS=linux GOARCH=amd64 go build -v -o build/scuttlego/linux-amd64/scuttlego./

        # Other platforms for scuttlego

        # Create zip archives (optional, but recommended)
	zip -r build/scuttlego/cmd/log-debugger-linux-amd64.zip build/scuttlego/cmd/log-debugger/linux-amd64
	zip -r build/scuttlego/cmd/log-debugger-darwin-amd64.zip build/scuttlego/cmd/log-debugger/darwin-amd64

        # If scuttlego is a main package:
	zip -r build/scuttlego-linux-amd64.zip build/scuttlego/linux-amd64
        # Other archives
