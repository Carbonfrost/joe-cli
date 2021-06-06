#: go engineering

# Automatically detect whether Go is in use
ENG_AUTODETECT_USING_GO = $(shell [ ! -f go.mod ] ; echo $$?)
ENG_AVAILABLE_RUNTIMES += go

# User can define ENG_USING_GO themselves to avoid autodeteciton
ifdef ENG_USING_GO
_ENG_ACTUALLY_USING_GO = $(ENG_USING_GO)
else
_ENG_ACTUALLY_USING_GO = $(ENG_AUTODETECT_USING_GO)
endif

.PHONY: \
	-ensure-go-output-dir \
	-go/build \
	-go/fmt \
	-go/get \
	-go/init \
	-hint-unsupported-go \
	-use/go-mod \
	go/build \
	go/fmt \
	go/get \
	go/init \
	use/go \

## Add support for Go to the project
use/go: | -go/init -use/go-mod

# Enable the tasks if we are using Go
ifeq (1,$(ENG_USING_GO))
ENG_ENABLED_RUNTIMES += go

GO_LDFLAGS ?= "-X f=unused"

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

_GO_OUTPUT_DIR=$(ENG_BUILD_DIR)/$(GOOS)_$(GOARCH)

## Install Go and project dependencies
go/init: -go/init

## Build Go project
go/build: -go/build

## Get dependent Go packages and modules
go/get: -go/get

## Format Go source files
go/fmt: -go/fmt

fetch: go/get
build: go/build
fmt: go/fmt

else
go/init: -hint-unsupported-go
go/build: -hint-unsupported-go
go/get: -hint-unsupported-go
go/fmt: -hint-unsupported-go
endif

-go/init:
	@    echo "$(_GREEN)Installing Go and Go dependencies...$(_RESET)"
	$(Q) $(OUTPUT_COLLAPSED) eng/brew_bundle_inject go
	$(Q) $(OUTPUT_COLLAPSED) brew bundle

-go/build: -ensure-go-output-dir
	$(Q) go build -ldflags="$(GO_LDFLAGS)" -o "$(_GO_OUTPUT_DIR)" ./...

-go/get:
	$(Q) go get ./...

-go/fmt:
	$(Q) go fmt ./...

-use/go-mod: -check-command-go
	$(Q) [ -f go.mod ] && $(OUTPUT_HIDDEN) go mod

-hint-unsupported-go:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
		"because $(_MAGENTA)go$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/go\`$(_RESET))"

-init-frameworks: go/init

-ensure-go-output-dir:
	$(Q) mkdir -p "$(_GO_OUTPUT_DIR)"
