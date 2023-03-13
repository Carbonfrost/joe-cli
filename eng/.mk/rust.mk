#: rust engineering

# Automatically detect whether Rust is in use
ENG_AUTODETECT_USING_RUST = $(shell [ ! -f Cargo.toml ] ; echo $$?)
ENG_AVAILABLE_STACKS += rust

RUST_SOURCE_FILES = $(shell find . -name '*.rs')

# User can define ENG_USING_RUST themselves to avoid autodeteciton
ifdef ENG_USING_RUST
_ENG_ACTUALLY_USING_RUST = $(ENG_USING_RUST)
else
_ENG_ACTUALLY_USING_RUST = $(ENG_AUTODETECT_USING_RUST)
endif

.PHONY: \
	-hint-unsupported-rust \
	-rust/build \
	-rust/fmt \
	-rust/init \
	-use/rust-cargo-toml \
	rust/build \
	rust/fmt \
	rust/init \
	use/rust \

## Add support for Rust to the project
use/rust: -rust/init

# Enable the tasks if we are using Rust
ifeq (1,$(ENG_USING_RUST))
ENG_ENABLED_STACKS += rust

## Install Rust and project dependencies
rust/init: -rust/init

## Build Rust project using Cargo
rust/build: -rust/build

## Remove Rust target directory
rust/clean: -rust/clean

## Format Rust source files
rust/fmt: -rust/fmt

build: rust/build
clean: rust/clean
fmt: rust/fmt

else
rust/init: -hint-unsupported-rust
rust/build: -hint-unsupported-rust
rust/clean: -hint-unsupported-rust
rust/fmt: -hint-unsupported-rust
endif

-rust/init:
	@    echo "$(_GREEN)Installing Rust and Rust dependencies...$(_RESET)"
	$(Q) $(OUTPUT_COLLAPSED) eng/brew_bundle_inject rustup
	$(Q) $(OUTPUT_COLLAPSED) brew bundle

-rust/build:
	$(Q) cargo build

-rust/clean:
	$(Q) cargo build

# It is possible there are no soruce files, so echo to rustfmt to prevent it
# looking for source from stdin
-rust/fmt: -check-command-rustfmt
	$(Q) echo | rustfmt $(RUST_SOURCE_FILES)

-hint-unsupported-rust:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
		"because $(_MAGENTA)rust$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/rust\`$(_RESET))"

-init-frameworks: rust/init
