# Copyright The HTNN Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL = /bin/bash
OS = $(shell uname)
IN_CI ?=

TARGET_SO       = libgolang.so
PROJECT_NAME    = mosn.io/htnn
# Both images use glibc 2.31. Ensure libc in the images match each other.
BUILD_IMAGE     ?= golang:1.21-bullseye
# This is the envoyproxy/envoy:contrib-v1.29.2
# Use docker inspect --format='{{index .RepoDigests 0}}' envoyproxy/envoy:contrib-v1.29.2
# to get the sha256 ID
PROXY_IMAGE     ?= envoyproxy/envoy@sha256:c47136604751274b30fa7a89132314b8e3586d54d8f8cc30d7a911a9ecc5e11c
# We don't use istio/proxyv2 because it is not designed to be run separately (need to work around permission issue).

# We may need to use timestamp if we need to update the image in one PR
DEV_TOOLS_IMAGE ?= ghcr.io/mosn/htnn-dev-tools:2024-03-05

VERSION   = $(shell cat VERSION)
GIT_VERSION     = $(shell git log -1 --pretty=format:%h)

MIN_K8S_VERSION = 1.26.0

# Define a recursive wildcard function
rwildcard=$(foreach d,$(wildcard $(addsuffix *,$(1))),$(call rwildcard,$d/,$(2))$(filter $(subst *,%,$(2)),$d))

PROTOC = protoc
PROTO_FILES = $(call rwildcard,./,*.proto)
GO_TARGETS = $(patsubst %.proto,%.pb.go,$(PROTO_FILES))

TEST_OPTION ?= -gcflags="all=-N -l" -race -covermode=atomic -coverprofile=cover.out -coverpkg=mosn.io/htnn/...

MOUNT_GOMOD_CACHE = -v $(shell go env GOPATH):/go
ifeq ($(IN_CI), true)
	# Mount go mod cache in the CI environment will cause 'Permission denied' error
	# when accessing files on host in later phase because the mounted directory will
	# have files which is created by the root user in Docker.
	# Run as low privilege user in the Docker doesn't
	# work because we also need root to create /.cache in the Docker.
	MOUNT_GOMOD_CACHE =
endif
