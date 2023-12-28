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


set -euo pipefail
set -x

os=$(uname)
os=${os,,} # convert to lowercase
platform=$(uname -m)
if [ "$platform" == "x86_64" ]; then
    platform="amd64"
fi

install-kubectl() {
    curl -L https://dl.k8s.io/release/"$KUBECTL_VERSION"/bin/"$os"/"$platform"/kubectl -o "$LOCATION"
    chmod +x "$LOCATION"
}

opt=$1
shift

${opt} "$@"
