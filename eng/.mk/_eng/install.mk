.PHONY: \
	-eng/install \
	-eng/.envrc \
	-eng/.editorconfig \

ENG_ENVRC_FILES = $(wildcard $(_ENG_BASE_DIR)/*.envrc)
ENG_ENVRC_FILES += $(foreach var,$(ENG_ENABLED_RUNTIMES),$(wildcard $(_ENG_RUNTIMES_DIR)/$(var)/*.envrc))

ENG_EDITORCONFIG_FILES = $(wildcard $(_ENG_BASE_DIR)/*.editorconfig)
ENG_EDITORCONFIG_FILES += $(foreach var,$(ENG_ENABLED_RUNTIMES),$(wildcard $(_ENG_RUNTIMES_DIR)/$(var)/*.editorconfig))

-eng/install: -eng/.envrc

# Generates .envrc by gluing together the preludes from the affected .envrc files
# that contain the documentation and then gluing together the rest of the scripts
-eng/.envrc:
	@ sed -n '/^[^#]/!p;//q' $(ENG_ENVRC_FILES) > .envrc
	@ awk '!/^[#]/'  $(ENG_ENVRC_FILES) >> .envrc

.envrc: -eng/.envrc
.editorconfig: -eng/.editorconfig

## Generate editor support files
eng/ready: -eng/.envrc -eng/.editorconfig

-eng/.editorconfig:	
	@ eng/editorconfig_merge $(ENG_EDITORCONFIG_FILES) > .editorconfig
