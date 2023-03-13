#: engineering platform help and debug

ifdef ALL
	_HELP_MAKEFILE_LIST = $(MAKEFILE_LIST)
else
	_DISABLED_STACKS = $(subst $(_SPACE),$(_PIPE),$(strip $(ENG_DISABLED_STACKS)))
	_HELP_MAKEFILE_LIST = $(shell echo "$(MAKEFILE_LIST)" | sed -E "s/[^ ]+($(_DISABLED_STACKS)).mk//g")
endif

.PHONY: help \
	list \
	doctor \

_AWK_VERSION = $(shell awk --version)

# Show help when no other goal is specified
.DEFAULT_GOAL = help

## Display pertinent environment variables
env: -env

## Show this help screen
help:
	@ echo "Engineering platform to support easier, polyglot development"
	@ if [[ "$(_AWK_VERSION)" == *"GNU Awk"* ]]; then \
		awk -f $(_ENG_MAKEFILE_DIR)/.mk/awk/makefile-help-screen.awk $(_HELP_MAKEFILE_LIST); \
	else \
		awk -f $(_ENG_MAKEFILE_DIR)/.mk/awk/makefile-simple-help-screen.awk $(_HELP_MAKEFILE_LIST) | sort; \
	fi
	@ echo
	@ echo "By default, targets from disabled frameworks are suppressed.  To review all targets, set ALL=1"

## List all targets
list:
	@ awk -f $(_ENG_MAKEFILE_DIR)/.mk/awk/makefile-list-targets.awk $(MAKEFILE_LIST) | grep -vE '^\.' | sort | uniq

## Version of engineering platform
version:
	@ echo "Engineering platform version:"
	$(Q) [[ -f "$(_ENG_VERSION_FILE)" ]] && paste -s $(_ENG_VERSION_FILE)

## Diagnose common issues
doctor: \
	-checks \
	-preflight-checks \

# "slow" checks used by doctor
-checks:

# "fast" checks that are implied on any nacho command
-preflight-checks:

# macOS-specific checks (or not)
-darwin-preflight-checks:
-non-darwin-preflight-checks:

ifeq ($(UNAME),Darwin)
-preflight-checks: -darwin-preflight-checks
else
-preflight-checks: -non-darwin-preflight-checks
endif
