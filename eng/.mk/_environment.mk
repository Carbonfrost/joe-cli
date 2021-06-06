.PHONY: \
	env \
	-env-global \
	-env-enabled-dotnet \
	-env-enabled-rust \
	-env-enabled-frameworks \
	-env-enabled-python \
	-env-enabled-ruby

_DEV_MESSAGE=(Is direnv set up correctly?  Have you tried 'make init'?)
-check-command-%:
	@ if [ ! $(shell command -v "${*}" ) ]; then \
		echo "Command ${*} could not be found $(_DEV_MESSAGE)"; \
		exit 1; \
	fi

-check-env-%:
	@ if [ "${${*}}" = "" ]; then \
		echo "Environment variable ${*} not set $(_DEV_MESSAGE)"; \
		exit 1; \
	fi

-env: -env-global -env-enabled-frameworks
	@ printf ""

-env-global:
	@ $(call _display_variables,ENG_GLOBAL_VARIABLES)
	@ if [[ -n "$(VERBOSE)" ]]; then \
		$(call _display_variables,ENG_GLOBAL_VERBOSE_VARIABLES) \
	fi

-env-enabled-frameworks: | -env-enabled-dotnet -env-enabled-python -env-enabled-ruby -env-enabled-go -env-enabled-rust

-env-enabled-dotnet:
	@ $(call _status,.NET,DOTNET)

-env-enabled-python:
	@ $(call _status,Python,PYTHON)

-env-enabled-ruby:
	@ $(call _status,Ruby,RUBY)

-env-enabled-go:
	@ $(call _status,Go,GO)

-env-enabled-rust:
	@ $(call _status,Rust,RUST)

# _status "display name" "base variable name"
define _status
	printf "$(_MAGENTA)%s$(_RESET) support is available and %b$(_RESET)\n" $(1) "$(if $(filter $(_ENG_ACTUALLY_USING_$(2)),1),$(_GREEN)enabled,$(_RED)not enabled)"
	if [[ -n "$(VERBOSE)" ]] || [[ $(_ENG_ACTUALLY_USING_$(2)) == "1" ]]; then \
	$(call _display_variables,ENG_$(2)_VARIABLES) \
	fi
	if [[ -n "$(VERBOSE)" ]]; then \
	$(call _display_variables,ENG_$(2)_VERBOSE_VARIABLES) \
	fi
endef

define _display_variables
	$(foreach var,$($(1)),printf "  $(_CYAN)%-22s$(_RESET) %s\n" $(var) "$($(var))";)
endef
