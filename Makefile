-include eng/Makefile

.DEFAULT_GOAL = build
.PHONY: \
	generate \
	watch \

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=-X 'github.com/Carbonfrost/gocli/internal/build.Version=$(BUILD_VERSION)'

build: generate

watch:
	@ find Makefile . -name '*.go' | entr -c gocli --version --plus --time generate

generate: -check-command-gucci
	$(Q) gucci -s Type=bool -s Name=Bool flag.go.tpl | gofmt > flag_bool.go
	$(Q) gucci -s Type=string -s Name=String flag.go.tpl | gofmt > flag_string.go
