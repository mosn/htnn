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
	_ "mosn.io/htnn/plugins/plugins/aicontentsecurity"
	_ "mosn.io/htnn/plugins/plugins/casbin"
	_ "mosn.io/htnn/plugins/plugins/celscript"
	_ "mosn.io/htnn/plugins/plugins/consumerrestriction"
	_ "mosn.io/htnn/plugins/plugins/debugmode"
	_ "mosn.io/htnn/plugins/plugins/demo"
	_ "mosn.io/htnn/plugins/plugins/extauth"
	_ "mosn.io/htnn/plugins/plugins/hmacauth"
	_ "mosn.io/htnn/plugins/plugins/keyauth"
	_ "mosn.io/htnn/plugins/plugins/limitToken"
	_ "mosn.io/htnn/plugins/plugins/limitcountredis"
	_ "mosn.io/htnn/plugins/plugins/limitreq"
	_ "mosn.io/htnn/plugins/plugins/oidc"
	_ "mosn.io/htnn/plugins/plugins/opa"
	_ "mosn.io/htnn/plugins/plugins/sentinel"
)
