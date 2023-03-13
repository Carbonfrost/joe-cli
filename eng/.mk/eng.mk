#: manage engineering platform itself

#
# Various commands for the Engineering platform itself
#

# Because these are used for installing eng itself, we can't depend on any of the utility targets
# or variables, so they are redefined here

_RESET = $(shell tput sgr0 2>/dev/null || printf '')
_RED = $(shell tput setaf 1 2>/dev/null || printf '')
_CYAN = $(shell tput setaf 6 2>/dev/null || printf '')

ifneq (, $(VERBOSE))
Q =
else
Q = @
endif

# This variable will be unset during the initial bootstrap.  So choose the
# default location based on working directory
_ENG_MAKEFILE_DIR := $(or $(_ENG_MAKEFILE_DIR),$(shell pwd)/eng)

_ENG_UPDATE_FILE := $(shell mktemp)
_ENG_VERSION_FILE := $(_ENG_MAKEFILE_DIR)/VERSION

ENG_DEV_UPDATE_REMOTE ?= file://$(HOME)/source/eng-commons-dotnet
ENG_UPDATE_BRANCH ?= master
ENG_GITHUB_REPO = https://github.com/Carbonfrost/eng-commons-dotnet

.PHONY: \
	-check-eng-updates-requirements \
	-clean-eng-directory \
	-download-eng-archive \
	-eng-update-start \
	-eng/start-Makefile \
	-generate-version-stub \
	eng/check \
	eng/enabled \
	eng/install \
	eng/start \
	eng/update \
	release/requirements \
	-eng/advice \

## Get started with eng, meant to be used in a new repo
eng/start: eng/update -eng/start-Makefile -eng/advice

## Display the names of active frameworks
eng/enabled:
	@ echo $(ENG_ENABLED_STACKS)

## Evaluate release requirements
release/requirements:
	$(Q) eng/release_requirements

eng/check:
	$(Q) eng/check_csproj

## Update eng platform itself to latest version
eng/update: -eng-update-start -download-eng-archive -clean-eng-directory
	$(Q) (tar -xf "$(_ENG_UPDATE_FILE)" --strip-components=1 'eng-commons-dotnet-$(ENG_UPDATE_BRANCH)/eng/*'; \
		tar -xf "$(_ENG_UPDATE_FILE)" --strip-components=2 'eng-commons-dotnet-$(ENG_UPDATE_BRANCH)/integration/*'; \
	)
	$(Q) echo $(ENG_UPDATE_BRANCH) > $(_ENG_VERSION_FILE)
	$(Q) git ls-remote $(_ENG_LS_REMOTE) refs/heads/$(ENG_UPDATE_BRANCH) | cut -f1 >> $(_ENG_VERSION_FILE)
	@ echo "Done! 🍺"

## Install integrations depending upon active stacks
eng/install: -eng/install

ifeq ($(ENG_DEV_UPDATE), 1)
-download-eng-archive: -check-eng-updates-requirements
	$(Q) git archive --format=zip --prefix=eng-commons-dotnet-$(ENG_UPDATE_BRANCH)/ --remote=$(ENG_DEV_UPDATE_REMOTE) $(ENG_UPDATE_BRANCH) -o $(_ENG_UPDATE_FILE)
-eng-update-start:
	@ echo "Installing engineering platform from dev $(ENG_DEV_UPDATE_REMOTE):$(ENG_UPDATE_BRANCH) ..."

_ENG_LS_REMOTE = $(ENG_DEV_UPDATE_REMOTE)

else
-download-eng-archive: -check-eng-updates-requirements
	$(Q) curl -o "$(_ENG_UPDATE_FILE)" -sL $(ENG_GITHUB_REPO)/archive/$(ENG_UPDATE_BRANCH).zip
-eng-update-start:
	@ echo "Installing engineering platform from $(ENG_UPDATE_BRANCH) ..."

_ENG_LS_REMOTE = $(ENG_GITHUB_REPO)

endif


-clean-eng-directory:
	$(Q) rm -rf eng

-check-eng-updates-requirements:
	@ if [ ! $(shell command -v curl ) ]; then \
		echo "$(_RED)fatal: $(_RESET)must have curl to download files"; \
		exit 1; \
	fi
	@ if [ ! $(shell command -v tar ) ]; then \
		echo "$(_RED)fatal: $(_RESET)must have tar to unarchive files"; \
		exit 1; \
	fi
	@ if [ ! $(shell command -v git ) ]; then \
		echo "$(_RED)fatal: $(_RESET)must have git to unarchive files"; \
		exit 1; \
	fi

-eng/start-Makefile:
	$(Q) printf -- "-include eng/Makefile\nhelp: -eng/advice\nstart:\n\t@ echo 'The Future awaits !'" > Makefile

-eng/advice:
	@ printf "Welcome to Carbonfrost!  "
	@ printf "To use a framework:\n\t$(_CYAN)make use/python$(_RESET)\n"
	@ printf "To install framework dependencies:\n\t$(_CYAN)make install/python$(_RESET)\n"
	@ printf "Try using the top-level target:\n\t$(_CYAN)make start$(_RESET)\n"
	@ printf "When ready, change the top-level target by adding this to Makefile:\n"
	@ printf "\t$(_CYAN).DEFAULT_GOAL = start$(_RESET)\n\n"
	@ printf "This causes the bare make command to run the top-level target.\n"