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
	_ "mosn.io/htnn/types/plugins/bandwidth_limit"
	_ "mosn.io/htnn/types/plugins/buffer"
	_ "mosn.io/htnn/types/plugins/casbin"
	_ "mosn.io/htnn/types/plugins/cel_script"
	_ "mosn.io/htnn/types/plugins/consumer_restriction"
	_ "mosn.io/htnn/types/plugins/cors"
	_ "mosn.io/htnn/types/plugins/debug_mode"
	_ "mosn.io/htnn/types/plugins/demo"
	_ "mosn.io/htnn/types/plugins/ext_auth"
	_ "mosn.io/htnn/types/plugins/ext_proc"
	_ "mosn.io/htnn/types/plugins/fault"
	_ "mosn.io/htnn/types/plugins/hmac_auth"
	_ "mosn.io/htnn/types/plugins/key_auth"
	_ "mosn.io/htnn/types/plugins/limit_count_redis"
	_ "mosn.io/htnn/types/plugins/limit_req"
	_ "mosn.io/htnn/types/plugins/local_ratelimit"
	_ "mosn.io/htnn/types/plugins/lua"
	_ "mosn.io/htnn/types/plugins/oidc"
	_ "mosn.io/htnn/types/plugins/opa"
)
