#
# Copyright (c) 2024 Peking University and Peking University
# Changsha Institute for Computing and Digital Economy
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
#

# Makefile for Golang components of CraneSched.
# This file will generate protobuf files, build executables and plugins.

# Notes for developer:
# - Please use proper indentation for neater output.


# Variables
#GIT_COMMIT_HASH := $(shell git rev-parse --short HEAD)
#VERSION_FILE := VERSION
#VERSION := $(shell [ -f $(VERSION_FILE) ] && cat $(VERSION_FILE) || echo $(GIT_COMMIT_HASH))
#BUILD_TIME := $(shell date +'%a, %d %b %Y %H:%M:%S %z')
#STRIP ?= false

GO_VERSION := $(shell go version)
GO_PATH := $(shell go env GOROOT)
GO := $(GO_PATH)/bin/go
COMMON_ENV := GOROOT=$(GO_PATH)
BUILD_FLAGS := -trimpath

NVIDIA_LIB_PATH := /usr/lib/x86_64-linux-gnu
CUDA_INCLUDE_PATH := /usr/include
PLUGIN_CGO_CFLAGS := -I$(CUDA_INCLUDE_PATH)
PLUGIN_CGO_LDFLAGS := -L$(NVIDIA_LIB_PATH) -lnvidia-ml -Wl,-rpath,$(NVIDIA_LIB_PATH)
CHECK_GPU := $(shell command -v nvidia-smi 2> /dev/null)

.PHONY: all protos build clean install plugin plugin-energy plugin-other

all: build
build: protos

protos:
	@echo "- Generating Protobuf files..."
	@mkdir -p ./generated/protos
	@protoc --go_out=paths=source_relative:generated/protos --go-grpc_out=paths=source_relative:generated/protos --proto_path=protos protos/*.proto
	@echo "  - Summary:"
	@echo "    - Protobuf files generated in ./generated/protos/"

