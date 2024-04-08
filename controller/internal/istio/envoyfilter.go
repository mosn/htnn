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

package istio

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/types/known/structpb"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mosn.io/htnn/api/pkg/plugins"
	ctrlcfg "mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/model"
)

func MustNewStruct(fields map[string]interface{}) *structpb.Struct {
	st, err := structpb.NewStruct(fields)
	if err != nil {
		// NewStruct returns error only when the fields contain non-standard type
		panic(err)
	}
	return st
}

const (
	DefaultHttpFilter = "htnn-http-filter"
	ECDSConsumerName  = "htnn-consumer"
)

type configWrapper struct {
	name   string
	pre    bool
	filter map[string]interface{}
}

func DefaultEnvoyFilters() map[string]*istiov1a3.EnvoyFilter {
	efs := map[string]*istiov1a3.EnvoyFilter{}

	defaultMatch := &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
		// As currently we only support Gateway cases, here we hardcode the context
		// to default envoy filters. We don't need to do that for user-defined envoy
		// filters. Because adding that will require a break change to remove it.
		// And user-defined envoy filter won't apply to mesh because:
		// 1. We don't support attaching policy to mesh.
		// 2. Mesh configuration doesn't have Go HTTP filter.
		ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
			Listener: &istioapi.EnvoyFilter_ListenerMatch{
				FilterChain: &istioapi.EnvoyFilter_ListenerMatch_FilterChainMatch{
					Filter: &istioapi.EnvoyFilter_ListenerMatch_FilterMatch{
						Name: "envoy.filters.network.http_connection_manager",
						SubFilter: &istioapi.EnvoyFilter_ListenerMatch_SubFilterMatch{
							Name: "envoy.filters.http.router",
						},
					},
				},
			},
		},
	}
	patches := []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{}
	// Go
	patches = append(patches, &istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
		ApplyTo: istioapi.EnvoyFilter_HTTP_FILTER,
		Match:   defaultMatch,
		Patch: &istioapi.EnvoyFilter_Patch{
			Operation: istioapi.EnvoyFilter_Patch_INSERT_FIRST,
			Value: MustNewStruct(map[string]interface{}{
				"disabled": true,
				"name":     "htnn.filters.http.golang",
				"typed_config": map[string]interface{}{
					"@type":        "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
					"library_id":   "fm",
					"library_path": ctrlcfg.GoSoPath(),
					"plugin_name":  "fm",
				},
			}),
		},
	})
	// Native filters can only be used before/after Go plugins.

	configs := []*configWrapper{}
	plugins.IterateHttpPlugin(func(key string, value plugins.Plugin) bool {
		nativePlugin, ok := value.(plugins.NativePlugin)
		if !ok {
			return true
		}

		filter := nativePlugin.HTTPFilterConfigPlaceholder()
		filter["name"] = fmt.Sprintf("htnn.filters.http.%s", key)
		filter["disabled"] = true
		configs = append(configs, &configWrapper{
			name:   key,
			pre:    nativePlugin.Order().Position == plugins.OrderPositionOuter,
			filter: filter,
		})
		return true
	})

	sort.Slice(configs, func(i, j int) bool {
		return plugins.ComparePluginOrder(configs[i].name, configs[j].name)
	})
	for _, config := range configs {
		filter := config.filter
		if config.pre {
			patches = append(patches, &istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: istioapi.EnvoyFilter_HTTP_FILTER,
				Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
					ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
						Listener: &istioapi.EnvoyFilter_ListenerMatch{
							FilterChain: &istioapi.EnvoyFilter_ListenerMatch_FilterChainMatch{
								Filter: &istioapi.EnvoyFilter_ListenerMatch_FilterMatch{
									Name: "envoy.filters.network.http_connection_manager",
									SubFilter: &istioapi.EnvoyFilter_ListenerMatch_SubFilterMatch{
										Name: "htnn.filters.http.golang",
									},
								},
							},
						},
					},
				},
				Patch: &istioapi.EnvoyFilter_Patch{
					Operation: istioapi.EnvoyFilter_Patch_INSERT_BEFORE,
					Value:     MustNewStruct(filter),
				},
			})
		} else {
			patches = append(patches, &istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: istioapi.EnvoyFilter_HTTP_FILTER,
				Match:   defaultMatch,
				Patch: &istioapi.EnvoyFilter_Patch{
					Operation: istioapi.EnvoyFilter_Patch_INSERT_BEFORE,
					Value:     MustNewStruct(filter),
				},
			})
		}
	}

	efs[DefaultHttpFilter] = &istiov1a3.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultHttpFilter,
		},
		Spec: istioapi.EnvoyFilter{
			ConfigPatches: patches,
			Priority:      100, // the priority is specific to silent the warning for relative operation
		},
	}

	return efs
}

func GenerateRouteFilter(host *model.VirtualHost, route string, config map[string]interface{}) *istiov1a3.EnvoyFilter {
	applyTo := istioapi.EnvoyFilter_HTTP_ROUTE
	vhost := &istioapi.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{
		Name: host.Name,
		Route: &istioapi.EnvoyFilter_RouteConfigurationMatch_RouteMatch{
			Name: route,
		},
	}

	return &istiov1a3.EnvoyFilter{
		Spec: istioapi.EnvoyFilter{
			ConfigPatches: []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: applyTo,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{
							RouteConfiguration: &istioapi.EnvoyFilter_RouteConfigurationMatch{
								Vhost: vhost,
							},
						},
					},
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_MERGE,
						Value: MustNewStruct(map[string]interface{}{
							"typed_per_filter_config": config,
						}),
					},
				},
			},
		},
	}
}

func GenerateConsumers(consumers map[string]interface{}) *istiov1a3.EnvoyFilter {
	return &istiov1a3.EnvoyFilter{
		Spec: istioapi.EnvoyFilter{
			ConfigPatches: []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: istioapi.EnvoyFilter_EXTENSION_CONFIG,
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_ADD,
						Value: MustNewStruct(map[string]interface{}{
							"name":     ECDSConsumerName,
							"disabled": true,
							"typed_config": map[string]interface{}{
								"@type":        "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
								"library_id":   "cm",
								"library_path": "/etc/libgolang.so",
								"plugin_name":  "cm",
								"plugin_config": map[string]interface{}{
									"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
									"value": consumers,
								},
							},
						}),
					},
				},
				{
					ApplyTo: istioapi.EnvoyFilter_HTTP_FILTER,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &istioapi.EnvoyFilter_ListenerMatch{
								FilterChain: &istioapi.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &istioapi.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
										SubFilter: &istioapi.EnvoyFilter_ListenerMatch_SubFilterMatch{
											Name: "envoy.filters.http.router",
										},
									},
								},
							},
						},
					},
					// We put the HTTP_FILTER in Consumer's patch, so that deployment which
					// doesn't use Consumer won't need to subscribe to this ECDS. The side effect
					// is that the first consumer will cause LDS drain, but it's similar to
					// deploy a Wasm plugin.
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_INSERT_BEFORE,
						Value: MustNewStruct(map[string]interface{}{
							"name": ECDSConsumerName,
							"config_discovery": map[string]interface{}{
								"type_urls": []interface{}{"type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config"},
								"config_source": map[string]interface{}{
									"ads": map[string]interface{}{},
								},
							},
						}),
					},
				},
			},
		},
	}
}
