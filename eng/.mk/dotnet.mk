#: dotnet engineering

#
# A standard system for building a dotnet assembly.  This should be included from the
# main Makefile
#
# Structure of the repository should be:
#
#    dotnet  (default root directory, configured with $ENG_DOTNET_DIR)
#    ├── Directory.Build.props   -- (Optional) Shared build attributes controlling version, etc.
#    ├── solution.sln
#    ├── src
#    │   └── Project1
#    │       ├── Automation  -- (Optional) String resources, text templates, etc.
#    │       │   ├── SR.properties
#    │       │   └── TextTemplates
#    │       ├── Project1.csproj
#    │       └── (source files)
#    └── test
#        └── Project1.UnitTests
#            ├── Project1.UnitTests.csproj
#            ├── Content    -- (Optional) Fixture files
#            │   └── ...
#            └── (source files)
#

ENG_AVAILABLE_RUNTIMES += dotnet

# Need to import or possibly re-import variables so that we have
# immediate evaluation context access to ENG_DOTNET_DIR, which controls
# autodetection later in this Make file
include $(dir $(lastword $(MAKEFILE_LIST)))/_variables.mk

.PHONY: \
	-check-dotnet-version \
	-dotnet/build \
	-dotnet/pack \
	-dotnet/publish \
	-dotnet/push \
	dotnet/build \
	dotnet/clean \
	dotnet/configure \
	dotnet/pack \
	dotnet/publish \
	dotnet/push \
	dotnet/restore \
	dotnet/test \

## Add support for .NET to the project
use/dotnet: | -dotnet/init -dotnet/solution

# Automatically detect whether .NET is in use.  This needs to be a static computation
# because it is used in the conditional below
ENG_AUTODETECT_USING_DOTNET := $(if $(wildcard $(ENG_DOTNET_DIR)/*.sln),1,0)

# User can define ENG_USING_DOTNET themselves to avoid autodeteciton
ifdef ENG_USING_DOTNET
_ENG_ACTUALLY_USING_DOTNET = $(ENG_USING_DOTNET)
else
_ENG_ACTUALLY_USING_DOTNET = $(ENG_AUTODETECT_USING_DOTNET)
endif

# Enable the tasks if we are using dotnet
ifeq (1,$(_ENG_ACTUALLY_USING_DOTNET))

ENG_ENABLED_RUNTIMES += dotnet
_ENG_ACTUAL_DOTNET_VERSION = $(dotnet --version)

## Install .NET  and project dependencies
dotnet/init: -dotnet/init dotnet/restore

## Set up dotnet configuration for NuGet
dotnet/configure: -dotnet/configure

## Restore package dependencies
dotnet/restore: -dotnet/restore

## Build the dotnet solution
dotnet/build: dotnet/restore -dotnet/build

## Executes dotnet clean
dotnet/clean: -dotnet/clean

## Pack the dotnet build into a NuGet package
dotnet/pack: dotnet/build -dotnet/pack

## Push the dotnet build into package repository
dotnet/push: dotnet/pack -dotnet/push

## Executes dotnet publish
dotnet/publish: dotnet/build -dotnet/publish

fetch: dotnet/restore
build: dotnet/build
redist: dotnet/redist
push: dotnet/push
clean: dotnet/clean
pack: dotnet/pack

else
dotnet/init: -hint-unsupported-dotnet
dotnet/configure: -hint-unsupported-dotnet
dotnet/restore: -hint-unsupported-dotnet
dotnet/build: -hint-unsupported-dotnet
dotnet/clean: -hint-unsupported-dotnet
dotnet/pack: -hint-unsupported-dotnet
dotnet/push: -hint-unsupported-dotnet
dotnet/publish: -hint-unsupported-dotnet
endif

-dotnet/configure: -requirements-dotnet -check-env-NUGET_SOURCE_NAME -check-env-NUGET_SOURCE_URL -check-env-NUGET_PASSWORD -check-env-NUGET_USER_NAME -check-env-NUGET_CONFIG_FILE
	$(Q) test -e $(NUGET_CONFIG_FILE) || echo "<configuration />" > $(NUGET_CONFIG_FILE)
	$(Q) $(OUTPUT_COLLAPSED) nuget sources add -Name $(NUGET_SOURCE_NAME) \
		-Source $(NUGET_SOURCE_URL) \
		-Password $(NUGET_PASSWORD) \
		-Username $(NUGET_USER_NAME) \
		-StorePasswordInClearText \
		-ConfigFile $(NUGET_CONFIG_FILE)

-dotnet/restore: -requirements-dotnet
	$(Q) $(OUTPUT_COLLAPSED) dotnet restore $(ENG_DOTNET_DIR)
	$(Q) $(OUTPUT_COLLAPSED) dotnet tool restore

-dotnet/clean:
	$(Q) rm $(_STANDARD_VERBOSE_FLAG) -rdf $(ENG_DOTNET_DIR)/{src,test}/*/{bin,obj}/

-dotnet/init:
	$(Q) $(OUTPUT_COLLAPSED) eng/brew_bundle_inject --cask dotnet-sdk
	$(Q) $(OUTPUT_COLLAPSED) brew bundle

-dotnet/solution: -requirements-dotnet
	$(Q) mkdir $(_STANDARD_VERBOSE_FLAG) -p $(ENG_DOTNET_DIR)/{src,test}
	$(Q) $(OUTPUT_COLLAPSED) dotnet new sln -o $(ENG_DOTNET_DIR) -n $(shell basename $$(pwd))

-dotnet/build: -requirements-dotnet -check-env-CONFIGURATION
	$(Q) eval $(shell eng/build_env); \
		$(OUTPUT_COLLAPSED) dotnet build --configuration $(CONFIGURATION) --no-restore $(ENG_DOTNET_DIR)

-dotnet/pack: -requirements-dotnet -check-env-CONFIGURATION
	$(Q) eval $(shell eng/build_env); \
		$(OUTPUT_COLLAPSED) dotnet pack --configuration $(CONFIGURATION) --no-build $(ENG_DOTNET_DIR)

-dotnet/publish: -requirements-dotnet -check-env-CONFIGURATION
	$(Q) eval $(shell eng/build_env); \
		$(OUTPUT_COLLAPSED) dotnet publish --configuration $(CONFIGURATION) --no-build $(ENG_DOTNET_DIR)

-dotnet/push: -requirements-dotnet -check-env-NUGET_PASSWORD -check-env-NUGET_USER_NAME -check-env-NUGET_UPLOAD_URL
	$(Q) for f in dotnet/src/*/bin/Release/*.nupkg; do \
		dotnet nuget push "$$f"; \
	done

-check-dotnet-version: -check-command-dotnet
	@ $(call _check_version,dotnet,$(_ENG_ACTUAL_DOTNET_VERSION),$(DOTNET_VERSION))

-requirements-dotnet: -check-dotnet-version -check-command-dotnet

-hint-unsupported-dotnet:
	@ echo $(_HIDDEN_IF_BOOTSTRAPPING) "$(_WARNING) Nothing to do" \
		"because $(_MAGENTA).NET$(_RESET) is not enabled (Investigate $(_CYAN)\`make use/dotnet\`$(_RESET))"

-init-frameworks: dotnet/init
