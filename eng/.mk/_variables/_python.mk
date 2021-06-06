# ------- Python settings
#

# Variables used by Python settings
ENG_PYTHON_VARIABLES := \
	PYENV_VERSION \
	VIRTUAL_ENV \
	VIRTUAL_ENV_NAME \

ENG_PYTHON_VERBOSE_VARIABLES := \
	VIRTUAL_ENV_DISABLE_PROMPT \
	PIP \
	PYTHON \

# Whether we are meant to use Python.  (See python.mk for autodetection)
ENG_USING_PYTHON ?= $(ENG_AUTODETECT_USING_PYTHON)

# Name of the python executable
PYTHON ?= python3

# Name of the pip executable
PIP ?= pip3

# Name of the virtual environment
VIRTUAL_ENV_NAME ?= venv
