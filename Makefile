-include eng/Makefile

.DEFAULT_GOAL = build
.PHONY: \
	generate \
	watch \
	lint \
	examples \
	coverage \
	coveragereport \
	install \
	-install-%

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=-X 'github.com/Carbonfrost/joe-cli/internal/build.Version=$(BUILD_VERSION)'

build: generate

watch:
	@ find Makefile . -name '*.go' | entr -c cli --version --plus --time generate

generate:
	$(Q) $(OUTPUT_COLLAPSED) go generate ./...

lint:
	$(Q) go vet ./... 2>&1 || true
	$(Q) go tool gocritic check ./... 2>&1 || true
	$(Q) go tool revive ./... 2>&1 || true
	$(Q) go tool staticcheck -checks 'all,-ST*' $(shell go list ./...) 2>&1	|| true

examples:
	$(Q) go build -o . ./_examples/joegit
	$(Q) go build -o . ./_examples/joefind
	$(Q) go build -o . ./_examples/joeopen

install: -install-joe

-install-%: build -check-env-PREFIX -check-env-_GO_OUTPUT_DIR
	$(Q) eng/install "${_GO_OUTPUT_DIR}/$*" $(PREFIX)/bin

coverage:
	$(Q) go test -coverprofile=coverage.txt -covermode=atomic ./...

coveragereport: coverage
	$(Q) go tool cover -html=coverage.txt
