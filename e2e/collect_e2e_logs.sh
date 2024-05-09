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


set -eo pipefail

collect() {
    ns=$1
    kubectl get pods -n "$ns" -o yaml > "$ns".log
    for pod in $(kubectl get pods -n "$ns" | awk '{print $1}' | grep -v "NAME"); do
        kubectl -n "$ns" logs "$pod" > "$pod"."$ns".log
    done
}

mkdir -p log
pushd log
collect istio-system
collect e2e
collect e2e-another
