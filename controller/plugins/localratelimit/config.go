// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package localratelimit

import (
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/localratelimit"
)

func init() {
	plugins.RegisterPlugin(localratelimit.Name, &plugin{})
}

type plugin struct {
	localratelimit.Plugin
}

// Each Native plugin need to implement the methods below

// ConfigTypeURL returns the type url of config
func (p *plugin) ConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit"
}

// Each HTTP Native plugin need to implement the methods below

// HTTPFilterConfigPlaceholder returns the placeholder config for http filter. The placeholder config
// will be validated but won't be used.
func (p *plugin) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":      p.ConfigTypeURL(),
			"statPrefix": "http_local_rate_limiter",
		},
	}
}
