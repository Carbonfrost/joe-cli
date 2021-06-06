.PHONY: \
	-eng/install \
	-eng/.envrc \

ENG_ENVRC_FILES = $(wildcard $(_ENG_BASE_DIR)/*.envrc)
ENG_ENVRC_FILES += $(foreach var,$(ENG_ENABLED_RUNTIMES),$(wildcard $(_ENG_RUNTIMES_DIR)/$(var)/*.envrc))

-eng/install: -eng/.envrc

# Generates .envrc by gluing together the preludes from the affected .envrc files
# that contain the documentation and then gluing together the rest of the scripts
-eng/.envrc:
	@ sed -n '/^[^#]/!p;//q' $(ENG_ENVRC_FILES) > .envrc
	@ awk '!/^[#]/'  $(ENG_ENVRC_FILES) >> .envrc

.envrc: -eng/.envrc
