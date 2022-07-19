# Most command output is silenced by default unless VERBOSE is set
VERBOSE ?=

# Prefix controlling where software is installed
PREFIX ?= /usr/local

# Redistributable package output directory
ENG_REDIST_DIR = redist

# Build output directory
ENG_BUILD_DIR = bin

# Provides a list of the enabled runtimes, derived from the ENG_USING_X variables
ENG_ENABLED_RUNTIMES +=

# Provides a list of the disabled runtimes, derived from the ENG_USING_X variables
ENG_DISABLED_RUNTIMES = $(filter-out $(ENG_ENABLED_RUNTIMES),$(ENG_AVAILABLE_RUNTIMES))

# Some variables that are globally interesting to examine in `make env`
ENG_GLOBAL_VARIABLES := \
	ENG_AVAILABLE_RUNTIMES \
	ENG_DISABLED_RUNTIMES \
	ENG_ENABLED_RUNTIMES \
	PATH \
	PREFIX \
	VERBOSE \
	BUILD_FIRST \

# Variables that are global but only show when VERBOSE is set
ENG_GLOBAL_VERBOSE_VARIABLES := \
	HOME \
	LANG \
	LC_CTYPE \
	TMPDIR \
	USER \
	DIRENV_DIR \
	ENG_REDIST_DIR \
	ENG_BUILD_DIR \

include $(_ENG_MAKEFILE_DIR)/.mk/_variables/*.mk

# -------
#
# `chronic` is a tool from moreutils which can suppress output except when
# errors occur.   If chronic is available, then OUTPUT_COLLAPSED can be used
# to suppress output conditionally
_CHRONIC = $(shell command -v chronic 2> /dev/null)

ifneq (, $(VERBOSE))
Q =
OUTPUT_HIDDEN =
OUTPUT_COLLAPSED =
_STANDARD_VERBOSE_FLAG =
else
Q = @
OUTPUT_HIDDEN = >/dev/null 2>/dev/null
OUTPUT_COLLAPSED = $(or $(_CHRONIC),$(OUTPUT_HIDDEN))
_STANDARD_VERBOSE_FLAG = -v
endif

ifneq (, $(DRY_RUN))
Q = @echo$(_SPACE)
endif

_DONE = echo "Done! 🍺" $(OUTPUT_HIDDEN)

# These variables are meant to be used internally

# Common escaped variables
_SPACE :=
_SPACE +=
_COMMA := ,
_PIPE := |

# Directories
_ENG_RUNTIMES_DIR = $(_ENG_MAKEFILE_DIR)/runtimes
_ENG_BASE_DIR = $(_ENG_MAKEFILE_DIR)/base

# Terminal output formatting
_RESET = $(shell tput sgr0 2>/dev/null || printf '')
_YELLOW = $(shell tput setaf 3 2>/dev/null || printf '')
_GREEN = $(shell tput setaf 2 2>/dev/null || printf '')
_RED = $(shell tput setaf 1 2>/dev/null || printf '')
_MAGENTA = $(shell tput setaf 5 2>/dev/null || printf '')
_CYAN = $(shell tput setaf 6 2>/dev/null || printf '')
_UNDERLINE = $(shell tput smul 2>/dev/null || printf '')
_BOLD = $(shell tput bold 2>/dev/null || printf '')

_FATAL_ERROR = $(_RED)fatal: $(_RESET)
_WARNING = $(_YELLOW)warning: $(_RESET)

# _check_version "command name" "actual version" "expected version"
define _check_version
	@ bash -c 'expected="$(3)"; \
	actual="$(2)"; \
	[[ $$actual == *$$expected* ]] || \
	printf >&2 "$(_WARNING)unexpected $(1) version $$actual (expected: $$expected)\n"'
endef
