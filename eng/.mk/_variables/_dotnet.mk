# ------- .NET settings
#

# Variables used by .NET settings
ENG_DOTNET_VARIABLES := \
	CONFIGURATION \
	DOTNET_VERSION \
	FRAMEWORK \
	NUGET_CONFIG_FILE \
	NUGET_SOURCE_URL \
	NUGET_UPLOAD_URL \
	NUGET_SOURCE_NAME \

ENG_DOTNET_VERBOSE_VARIABLES := \
	NUGET_USER_NAME \
	NUGET_PASSWORD \

# Directory to use as root of a dotnet project.
ENG_DOTNET_DIR := ./dotnet

# The location of the NuGet configuration file
NUGET_CONFIG_FILE ?= ./nuget.config

# The configuration to build (probably "Debug" or "Release")
CONFIGURATION ?= Release

# The framework to publish
FRAMEWORK ?= netcoreapp3.0

# Desired version of .NET to use
DOTNET_VERSION ?= 3.1
