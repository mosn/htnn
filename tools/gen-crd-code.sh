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

readonly REPO=mosn.io/htnn
readonly OUTPUT_PKG=${REPO}/types/pkg/client
readonly APIS_PKG=${REPO}/types
readonly CLIENTSET_NAME=versioned
readonly CLIENTSET_PKG_NAME=clientset
readonly GOPATH="$(mktemp -d)"
readonly SCRIPT_ROOT="$(pwd)/types"

export GOPATH
mkdir -p "$GOPATH/src/$REPO"
ln -s "${SCRIPT_ROOT}" "$GOPATH/src/$APIS_PKG"

if [[ "${VERIFY_CODEGEN:-}" == "true" ]]; then
    echo "Running in verification mode"
    readonly VERIFY_FLAG="--verify-only"
fi

readonly COMMON_FLAGS="${VERIFY_FLAG:-} --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt"

echo "Generating clientset at ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME}"
"${LOCALBIN}"/client-gen \
    --clientset-name "${CLIENTSET_NAME}" \
    --input-base "" \
    --input "${APIS_PKG}/apis/v1" \
    --output-package "${OUTPUT_PKG}/${CLIENTSET_PKG_NAME}" \
    ${COMMON_FLAGS}
