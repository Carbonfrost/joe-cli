#: python engineering

ENG_AUTODETECT_USING_PYTHON = $(shell \
	[ ! -f requirements.txt ] && \
	[ ! -f Pipfile ] ; \
	echo $$? \
)

# User can define ENG_USING_PYTHON themselves to avoid autodeteciton
ifdef ENG_USING_PYTHON
_ENG_ACTUALLY_USING_PYTHON = $(ENG_USING_PYTHON)
else
_ENG_ACTUALLY_USING_PYTHON = $(ENG_AUTODETECT_USING_PYTHON)
endif

ENG_AVAILABLE_STACKS += python

_VENV = . $(VIRTUAL_ENV_NAME)/bin/activate $(OUTPUT_HIDDEN)
_ENG_ACTUAL_PYTHON_VERSION=$(shell $(PYTHON) -V | awk '{ print $$2 }')

.PHONY: \
	-check-python-version \
	-hint-unsupported-python \
	-python/init \
	-python/install \
	-requirements-python \
	-use/python \
	-use/python-Pipfile \
	python/init \
	python/install \
	use/python \

## Add support for python
use/python: | -python/init -use/python-Pipfile

fetch: python/install

# Enable the tasks if we are using python
ifeq (1, $(ENG_USING_PYTHON))

ENG_ENABLED_STACKS += python

## Install python and project dependencies
python/init: -python/init

## Install requirements using pip and pipenv
python/install: -python/install

else
python/init: -hint-unsupported-python
python/install: -hint-unsupported-python
endif

-python/init: -check-command-brew -python/init-python -python/init-venv -python/install

-use/python-Pipfile: -check-command-pipenv
	$(Q) [ -f Pipfile ] || ( \
		$(_VENV) && $(OUTPUT_HIDDEN) pipenv lock \
	)

-hint-unsupported-python:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
		"because $(_MAGENTA)python$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/python\`$(_RESET))"

-python/init-python:
	@    echo "$(_GREEN)Installing Python and Python dependencies...$(_RESET)"
	$(Q) $(OUTPUT_COLLAPSED) eng/brew_bundle_inject $(PYTHON) pipenv
	$(Q) $(OUTPUT_COLLAPSED) brew bundle

-python/init-venv: -check-command-$(PYTHON)
	$(Q) if [ "$(VIRTUAL_ENV)" != "$(shell pwd)/$(VIRTUAL_ENV_NAME)" ] || [ ! -d "$(VIRTUAL_ENV)" ]; then \
		$(OUTPUT_HIDDEN) $(PYTHON) -m venv $(VIRTUAL_ENV_NAME); \
	fi
	$(Q) ( \
		$(_VENV) && \
		$(OUTPUT_HIDDEN) $(PIP) install pipenv \
	)

-python/install: -python/install-pip -python/install-pipenv

-python/install-pip: -check-command-$(PYTHON)
	$(Q) if [ -f requirements.txt ]; then \
		$(_VENV) && $(OUTPUT_HIDDEN) $(PYTHON) -m pip install; \
	fi

-python/install-pipenv: -python/init-venv
	$(Q) if [ -f Pipfile ]; then \
		$(_VENV) && command -v pipenv > /dev/null && $(OUTPUT_HIDDEN) pipenv install; \
	fi

-check-python-version: -check-command-python
	@ $(call _check_version,python,$(_ENG_ACTUAL_PYTHON_VERSION),$(PYTHON_VERSION))

-requirements-python: -check-python-version -check-command-python

-init-frameworks: python/init
