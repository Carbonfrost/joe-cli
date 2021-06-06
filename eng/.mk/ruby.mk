#: ruby engineering

# Automatically detect whether Ruby is in use
ENG_AUTODETECT_USING_RUBY = $(shell [ ! -f .ruby-version ] ; echo $$?)

# User can define ENG_USING_RUBY themselves to avoid autodeteciton
ifdef ENG_USING_RUBY
_ENG_ACTUALLY_USING_RUBY = $(ENG_USING_RUBY)
else
_ENG_ACTUALLY_USING_RUBY = $(ENG_AUTODETECT_USING_RUBY)
endif

ENG_AVAILABLE_RUNTIMES += ruby

.PHONY: \
	-hint-unsupported-ruby \
	-ruby/init \
	-use/ruby \
	-use/ruby-Gemfile \
	-use/ruby-version \
	ruby/init \
	use/ruby \

## Add support for Ruby to the project
use/ruby: | -use/ruby-version -ruby/init -use/ruby-Gemfile

# Enable the tasks if we are using ruby
ifeq (1,$(ENG_USING_RUBY))
ENG_ENABLED_RUNTIMES += ruby

## Install Ruby and project dependencies
ruby/init: -ruby/init
else
ruby/init: -hint-unsupported-ruby
endif

-ruby/init: -check-command-rbenv
	@    echo "$(_GREEN)Installing Ruby and Ruby dependencies...$(_RESET)"
	$(Q) $(OUTPUT_COLLAPSED) rbenv install -s
	$(Q) $(OUTPUT_COLLAPSED) gem install bundler
	$(Q) [ -f Gemfile ] && $(OUTPUT_HIDDEN) bundle install

-use/ruby-Gemfile: -check-command-Gemfile
	$(Q) [ -f Gemfile ] || bundle init

-use/ruby-version:
	@    echo "Adding support for Ruby to this project... "
	$(Q) [ -f .ruby-version ] || echo $(ENG_LATEST_RUBY_VERSION) > .ruby-version

-hint-unsupported-ruby:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
		"because $(_MAGENTA)Ruby$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/ruby\`$(_RESET))"

-init-frameworks: ruby/init
