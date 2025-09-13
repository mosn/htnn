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
	_ "mosn.io/htnn/types/dynamicconfigs"
	_ "mosn.io/htnn/types/plugins/aicontentsecurity"
	_ "mosn.io/htnn/types/plugins/bandwidthlimit"
	_ "mosn.io/htnn/types/plugins/buffer"
	_ "mosn.io/htnn/types/plugins/casbin"
	_ "mosn.io/htnn/types/plugins/celscript"
	_ "mosn.io/htnn/types/plugins/consumerrestriction"
	_ "mosn.io/htnn/types/plugins/cors"
	_ "mosn.io/htnn/types/plugins/debugmode"
	_ "mosn.io/htnn/types/plugins/demo"
	_ "mosn.io/htnn/types/plugins/extauth"
	_ "mosn.io/htnn/types/plugins/extproc"
	_ "mosn.io/htnn/types/plugins/fault"
	_ "mosn.io/htnn/types/plugins/hmacauth"
	_ "mosn.io/htnn/types/plugins/keyauth"
	_ "mosn.io/htnn/types/plugins/limitToken"
	_ "mosn.io/htnn/types/plugins/limitcountredis"
	_ "mosn.io/htnn/types/plugins/limitreq"
	_ "mosn.io/htnn/types/plugins/listenerpatch"
	_ "mosn.io/htnn/types/plugins/localratelimit"
	_ "mosn.io/htnn/types/plugins/lua"
	_ "mosn.io/htnn/types/plugins/networkrbac"
	_ "mosn.io/htnn/types/plugins/oidc"
	_ "mosn.io/htnn/types/plugins/opa"
	_ "mosn.io/htnn/types/plugins/routepatch"
	_ "mosn.io/htnn/types/plugins/sentinel"
	_ "mosn.io/htnn/types/plugins/tlsinspector"
)
