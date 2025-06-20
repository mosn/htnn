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

package test

import input.request

default allow = false

allow {
    request.method == "GET"
    startswith(request.path, "/echo")
}

custom_response = {
    "body": "Authentication required. Please provide valid authorization header.",
    "status_code": 401,
    "headers": {
        "WWW-Authenticate": ["Bearer realm=\"api\""],
        "Content-Type": ["application/json"]
    }
} {
    request.method == "GET"
    startswith(request.path, "/x")
}
