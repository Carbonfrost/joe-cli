#!/usr/bin/env bash
project_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

# Build a .NET executable if necessary
function _build_first(){
  local app_name=$1

  if [[ -n "$BUILD_FIRST" ]]; then
    echo >&2 "Rebuilding $app_name... (see BUILD_FIRST in .envrc)"
    pushd "$project_dir" > /dev/null || exit 1
      make dotnet/build >&2
    popd > /dev/null || exit 1
  fi
}

# Execute a .NET executable
function _exec(){
  local app_name=$1
  local opts
  opts=$(tr "[:lower:]" "[:upper:]" <<< "${app_name}_OPTIONS")
  shift

  CONFIGURATION="Release"
  PROFILE="netcoreapp3.0"
  BINARY="$project_dir/dotnet/src/$app_name/bin/$CONFIGURATION/$PROFILE/$app_name.dll"
  dotnet $DOTNET_OPTIONS "$BINARY" ${!opts} $@
}
