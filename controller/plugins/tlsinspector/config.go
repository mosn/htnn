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

package tlsinspector

import (
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/tlsinspector"
)

func init() {
	plugins.RegisterPlugin(tlsinspector.Name, &plugin{})
}

type plugin struct {
	tlsinspector.Plugin
}

func (p *plugin) ConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector"
}
