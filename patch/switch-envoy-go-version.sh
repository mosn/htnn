#!/usr/bin/env bash
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

# This script is used to switch the Envoy version in go.mod files.
# It replaces, rather than accumulates, the Envoy replace directive.
set -euo pipefail

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <envoy_version>"
    exit 1
fi

envoy_version=$1

if [[ ! "$envoy_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Envoy version $envoy_version should be in the format of x.y.z"
    exit 1
fi

if [[ "$envoy_version" =~ ^1\.(35|36|37|38|39)\.[0-9]+$ ]]; then
    # Envoy 1.35+ selects a compatible build-tag implementation.
    :
fi

if [[ ! "$envoy_version" =~ ^1\.(29|31|32|33|35|36|37|38|39)\.[0-9]+$ ]]; then
    echo "Unsupported envoy version $envoy_version"
    exit 1
fi

set_envoy_replace() {
    target_file=$1
    replacement="replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v$envoy_version"

    if grep -q '^replace github.com/envoyproxy/envoy ' "$target_file"; then
        tmp_file=$(mktemp)
        grep -v '^replace github.com/envoyproxy/envoy ' "$target_file" | awk 'NF || !previous_blank { print } { previous_blank = !NF }' > "$tmp_file"
        mv "$tmp_file" "$target_file"
    fi
    printf '\n%s\n' "$replacement" >> "$target_file"
    echo "Set Envoy replace in $target_file to v$envoy_version"
}

remove_envoy_replace() {
    target_file=$1
    if grep -q '^replace github.com/envoyproxy/envoy ' "$target_file"; then
        tmp_file=$(mktemp)
        grep -v '^replace github.com/envoyproxy/envoy ' "$target_file" > "$tmp_file"
        mv "$tmp_file" "$target_file"
        echo "Removed Envoy replace from $target_file"
    fi
}

require_if_missing() {
    requirement=$1
    target_file=$2
    if ! grep -qxF "$requirement" "$target_file"; then
        printf '\n%s\n' "$requirement" >> "$target_file"
        echo "Added '$requirement' to $target_file"
    fi
}

remove_contrib_requirement() {
    target_file=$1
    if grep -q '^require github.com/envoyproxy/go-control-plane/contrib ' "$target_file"; then
        tmp_file=$(mktemp)
        grep -v '^require github.com/envoyproxy/go-control-plane/contrib ' "$target_file" | awk 'NF || !previous_blank { print } { previous_blank = !NF }' > "$tmp_file"
        mv "$tmp_file" "$target_file"
        echo "Removed go-control-plane/contrib requirement from $target_file"
    fi
}

if [[ "$envoy_version" =~ ^1\.32\.[0-9]+$ ]]; then
    # The default module requirement already uses Envoy 1.32.
    remove_envoy_replace api/go.mod
    remove_envoy_replace plugins/go.mod
    remove_contrib_requirement api/go.mod
    remove_contrib_requirement plugins/go.mod
    exit 0
fi

# Maintaining a replace is easier than maintaining a go.mod file for -modfile.
set_envoy_replace api/go.mod
set_envoy_replace plugins/go.mod

# Envoy 1.37+ depends on go-control-plane/envoy submodule (split from the monolithic
# go-control-plane). The contrib/ directory (v3alpha packages) moved to its own submodule:
# go-control-plane/contrib. We require it explicitly so the v3alpha imports resolve.
if [[ "$envoy_version" =~ ^1\.(3[7-9]|[4-9]) ]] || [[ "$envoy_version" =~ ^0\.0\.0 ]]; then
    require_if_missing "require github.com/envoyproxy/go-control-plane/contrib v1.36.0" api/go.mod
    require_if_missing "require github.com/envoyproxy/go-control-plane/contrib v1.36.0" plugins/go.mod
else
    remove_contrib_requirement api/go.mod
    remove_contrib_requirement plugins/go.mod
fi
