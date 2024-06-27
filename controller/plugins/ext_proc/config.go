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

package ext_proc

import (
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/ext_proc"
)

func init() {
	plugins.RegisterPlugin("outerExtProc", &outerPlugin{})
	plugins.RegisterPlugin("innerExtProc", &innerPlugin{})
}

type plugin struct {
}

func (p *plugin) ToRouteConfig(config map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		// v3.ExtProcOverrides -> v3.ExtProcPerRoute
		"overrides": config,
		// TODO: for now, we don't have a way to generate the cluster used in the ExtProcPerRoute's grpcService field.
		// User need to manually add the cluster via Service or ServiceEntry
		// Maybe we can add a wrapper to generate the cluster name according to user's configuration, like
		// https://github.com/alibaba/higress/blob/a787088c0e2a1cd76c5f8e3e92b6e05bc3a85d9a/plugins/wasm-go/pkg/wrapper/cluster_wrapper.go#L56.
		// Generate a cluster via EnvoyFilter is not a good idea to me, because it's not the plugin's duty to maintain the lifecycle
		// of the cluster.
	}
}

func (p *plugin) ConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.ext_proc.v3.ExtProcPerRoute"
}

func (p *plugin) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type": "type.googleapis.com/envoy.extensions.filters.http.ext_proc.v3.ExternalProcessor",
			"grpcService": map[string]interface{}{
				"envoy_grpc": map[string]interface{}{
					// This is a cluster generated by istio to dispatch xDS. We need to a real
					// cluster here to satisfy the Envoy validation. It won't be used to handle ext_proc.
					"cluster_name": "xds-grpc",
				},
			},
		},
	}
}

type outerPlugin struct {
	plugin
	ext_proc.OuterPlugin
}

type innerPlugin struct {
	plugin
	ext_proc.InnerPlugin
}
