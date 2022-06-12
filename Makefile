-include eng/Makefile

.DEFAULT_GOAL = build
.PHONY: \
	generate \
	watch \
	lint \
	examples \

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=-X 'github.com/Carbonfrost/joe-cli/internal/build.Version=$(BUILD_VERSION)'

build: generate

watch:
	@ find Makefile . -name '*.go' | entr -c cli --version --plus --time generate

generate:
	$(Q) go generate ./...

lint:
	$(Q) go run honnef.co/go/tools/cmd/staticcheck -checks 'all,-ST*' $(shell go list ./...)

examples:
	$(Q) go build -o . ./examples/joegit
	$(Q) go build -o . ./examples/joefind
	$(Q) go build -o . ./examples/joeopen
