#: use homebrew to manage macOS dependencies

ENG_AUTODETECT_USING_HOMEBREW_BUNDLE = $(shell [ ! -f Brewfile ] ; echo $$?)
ENG_USING_HOMEBREW_BUNDLE ?= $(ENG_AUTODETECT_USING_HOMEBREW_BUNDLE)

.PHONY: homebrew/init

## Use Homebrew bundle in this project to manage dependencies
use/homebrew: | -use/homebrew-bundle

ifeq (1, $(ENG_USING_HOMEBREW_BUNDLE))

## Install software for developing on macOS
homebrew/init: | -homebrew-init
else
homebrew/init: -hint-unsupported-homebrew-bundle
endif

-homebrew-init: -check-command-brew
	@    echo "$(_GREEN)Installing macOS dependencies...$(_RESET) (You may be prompted for a password)"
	$(Q) $(OUTPUT_COLLAPSED) brew tap homebrew/bundle
	$(Q) $(OUTPUT_COLLAPSED) brew bundle

-use/homebrew-bundle: -check-command-brew
	$(Q) [ -f Brewfile ] || $(OUTPUT_COLLAPSED) brew bundle dump

-hint-unsupported-homebrew-bundle:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
		"because $(_MAGENTA)Homebrew bundle$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/homebrew\`$(_RESET))"
