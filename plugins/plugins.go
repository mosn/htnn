// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugins

import (
	_ "mosn.io/htnn/plugins/plugins/casbin"
	_ "mosn.io/htnn/plugins/plugins/cel_script"
	_ "mosn.io/htnn/plugins/plugins/consumer_restriction"
	_ "mosn.io/htnn/plugins/plugins/debug_mode"
	_ "mosn.io/htnn/plugins/plugins/demo"
	_ "mosn.io/htnn/plugins/plugins/ext_auth"
	_ "mosn.io/htnn/plugins/plugins/hmac_auth"
	_ "mosn.io/htnn/plugins/plugins/key_auth"
	_ "mosn.io/htnn/plugins/plugins/limit_count_redis"
	_ "mosn.io/htnn/plugins/plugins/limit_req"
	_ "mosn.io/htnn/plugins/plugins/oidc"
	_ "mosn.io/htnn/plugins/plugins/opa"
)
