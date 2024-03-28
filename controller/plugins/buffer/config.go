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

package buffer

import (
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/buffer"
)

const (
	Name = "buffer"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	buffer.Plugin
}

// The BufferPerRoute has two fields: `disabled` and `buffer`:
// https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/buffer/v3/buffer.proto#extensions-filters-http-buffer-v3-bufferperroute
// The `disabled` is useless for us. And it's ugly to use another `buffer` field in the `buffer` plugin.
// So here we introduce a conversion function to make the configuration more friendly.
func (p *plugin) ToRouteConfig(config map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		// v3.Buffer -> v3.BufferPerRoute
		"buffer": config,
	}
}

func (p *plugin) RouteConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.buffer.v3.BufferPerRoute"
}

func (p *plugin) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":           "type.googleapis.com/envoy.extensions.filters.http.buffer.v3.Buffer",
			"maxRequestBytes": 42,
		},
	}
}
