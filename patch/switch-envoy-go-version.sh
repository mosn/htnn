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

# This script is used to switch the envoy version in go.mod files
set -euo pipefail

envoy_version=$1

# Check if arguments were provided
if [ -z "$envoy_version" ]; then
    echo "Usage: $0 <envoy_version>"
    exit 1
fi

if [[ ! "$envoy_version" =~ ^[0-9]+\.[0-9]+\. ]]; then
    echo "Envoy version $envoy_version should be in the format of x.y.z"
    exit 1
fi

if [[ "$envoy_version" =~ ^1\.32\.[0-9]+$ ]]; then
    # patch version should not matter
    echo "Envoy version $envoy_version is already in used"
    exit 0
fi

if [[ ! "$envoy_version" =~ ^1\.(29|31|32|33)\. ]]; then
    echo "Unsupported envoy version $envoy_version"
    exit 1
fi

append_if_need() {
    search_string=$1
    target_file=$2
    # Search for the string in the file
    if ! grep -q "$search_string" "$target_file"; then
        # Append the string to the file
        echo "$search_string" >> "$target_file"
        echo "String '$search_string' appended to $target_file"
    else
        echo "String '$search_string' already exists in $target_file"
    fi
}

# append to go.mod is easier than maintaining a go.mod file for -modfile flag
append_if_need "replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v$envoy_version" api/go.mod
append_if_need "replace github.com/envoyproxy/envoy => github.com/envoyproxy/envoy v$envoy_version" plugins/go.mod
