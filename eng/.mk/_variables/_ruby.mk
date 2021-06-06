# ------- Ruby settings
#

# Variables used by Ruby settings
ENG_RUBY_VARIABLES := \
	RBENV_SHELL \
	RUBY_PATH \

ENG_RUBY_VERBOSE_VARIABLES := \
     RUBY_GC_HEAP_INIT_SLOTS \
     RUBY_GC_HEAP_FREE_SLOTS \
     RUBY_GC_HEAP_GROWTH_FACTOR \
     RUBY_GC_HEAP_GROWTH_MAX_SLOTS \

# Whether we are meant to use Ruby.  (See ruby.mk for autodetection)
ENG_USING_RUBY ?= $(ENG_AUTODETECT_USING_RUBY)

# Latest version of Ruby supported
ENG_LATEST_RUBY_VERSION = 2.6.0
