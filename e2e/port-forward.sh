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


set -eou pipefail
set -x

DEST="$1"
if [[ "$DEST" == "istio-ingressgateway" ]]; then
    exec kubectl port-forward -n istio-system pod/"$(kubectl -n istio-system get pods | grep '^istio-ingressgateway' |  cut -d' ' -f 1)" 18000:18000
else
    exec kubectl port-forward -n e2e pod/"$(kubectl -n e2e get pods | grep '^default-istio' |  cut -d' ' -f 1)" 10000:10000
fi
