#: execute project commands across frameworks

.PHONY: \
	-direnv-install \
	-homebrew-install \
	-init-frameworks \
	build \
	clean \
	default \
	fetch \
	init \
	install \

## Fetch dependencies
fetch: _HIDDEN_IF_BOOTSTRAPPING=>/dev/null
fetch:
	@ $(_DONE)

## Build project outputs
build: fetch
	@ $(_DONE)

## Install build outputs so they are ready to use on this computer
install: build
	@ $(_DONE)

## Clean up intermediate build targets
clean:
	@ $(_DONE)

## Create packages for various online package repositories
pack:
	@ $(_DONE)

## Format source code
fmt:
	@ $(_DONE)

# To simplify things, we just try to init every framework even if
# they are not enabled.  However, we don't want to display hint messages
# about them not being enabled, so that's what this variable is used for

## Initialize dependencies for developing the project
init:
	@ echo "Installing prerequisites and setting up the environment ..."
	@ $(MAKE) VERBOSE=$(VERBOSE) _HIDDEN_IF_BOOTSTRAPPING=">/dev/null" \
		-- -homebrew-install -direnv-install -init-frameworks
	@ echo "Done! 🍺"

# At a minimum, we need to install Homebrew to manage dependencies.  We
# at least need direnv as a dependency
# However, other brews and casks are opt-in via the use of Brewfile.
-homebrew-install:
	@ if ! command -v "brew" > /dev/null ; then \
		echo >&2 "$(_FATAL_ERROR) Please install Homebrew first (https://brew.sh)"; \
		exit 1; \
	fi
	$(Q) $(OUTPUT_COLLAPSED) brew update

-direnv-install: -check-command-brew
	$(Q) $(OUTPUT_COLLAPSED) eng/brew_bundle_inject direnv
	$(Q) $(OUTPUT_COLLAPSED) brew bundle
	$(Q) $(OUTPUT_COLLAPSED) direnv allow

-init-frameworks:
