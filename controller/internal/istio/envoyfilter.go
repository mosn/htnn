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
	"encoding/json"
	"fmt"
	"sort"

	"google.golang.org/protobuf/types/known/structpb"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fmModel "mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/pkg/plugins"
	ctrlcfg "mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/model"
	"mosn.io/htnn/controller/pkg/component"
	"mosn.io/htnn/controller/pkg/constant"
	mosniov1 "mosn.io/htnn/types/apis/v1"
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
	DefaultHTTPFilter            = "htnn-http-filter"
	ECDSConsumerName             = "htnn-consumer"
	DynamicConfigEnvoyFilterName = "htnn-dynamic-config"
)

type configWrapper struct {
	name   string
	pre    bool
	filter map[string]interface{}
}

func DefaultEnvoyFilters() map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter {
	efs := map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter{}

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
	plugins.IteratePlugin(func(key string, value plugins.Plugin) bool {
		nativePlugin, ok := value.(plugins.HTTPNativePlugin)
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

	key := component.EnvoyFilterKey{
		Namespace: ctrlcfg.RootNamespace(),
		Name:      DefaultHTTPFilter,
	}
	efs[key] = &istiov1a3.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctrlcfg.RootNamespace(),
			Name:      DefaultHTTPFilter,
			Labels: map[string]string{
				constant.LabelCreatedBy: "FilterPolicy",
			},
		},
		Spec: istioapi.EnvoyFilter{
			ConfigPatches: patches,
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
		// We don't set ObjectMeta here because this EnvoyFilter will be merged later
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

func GenerateLDSFilter(key string, ldsName string, hasHCM bool, config map[string]interface{}) *istiov1a3.EnvoyFilter {
	ef := &istiov1a3.EnvoyFilter{
		Spec: istioapi.EnvoyFilter{},
	}

	if config[model.CategoryListener] != nil {
		cfg, _ := config[model.CategoryListener].([]*fmModel.FilterConfig)
		for _, filter := range cfg {
			c, _ := filter.Config.(map[string]interface{})
			ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches,
				&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
					ApplyTo: istioapi.EnvoyFilter_LISTENER,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &istioapi.EnvoyFilter_ListenerMatch{
								Name: ldsName,
							},
						},
					},
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_MERGE,
						Value:     MustNewStruct(c),
					},
				},
			)
		}
	}

	if config[model.CategoryECDSListener] != nil {
		cfg, _ := config[model.CategoryECDSListener].([]*fmModel.FilterConfig)
		for i := len(cfg) - 1; i >= 0; i-- {
			filter := cfg[i]
			ecdsName := key + "-" + filter.Name
			c, _ := filter.Config.(map[string]interface{})
			typeURL, _ := c["@type"].(string)
			ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches,
				&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
					ApplyTo: istioapi.EnvoyFilter_LISTENER_FILTER,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &istioapi.EnvoyFilter_ListenerMatch{
								Name: ldsName,
							},
						},
					},
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_INSERT_FIRST,
						Value: MustNewStruct(map[string]interface{}{
							"name": ecdsName,
							"config_discovery": map[string]interface{}{
								"config_source": map[string]interface{}{
									"ads": map[string]interface{}{},
								},
								"type_urls": []interface{}{
									typeURL,
								},
							},
						}),
					},
				},
				&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
					ApplyTo: istioapi.EnvoyFilter_EXTENSION_CONFIG,
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_ADD,
						Value: MustNewStruct(map[string]interface{}{
							"name":         ecdsName,
							"typed_config": filter.Config,
						}),
					},
				},
			)
		}
	}

	if config[model.CategoryECDSNetwork] != nil {
		cfg, _ := config[model.CategoryECDSNetwork].([]*fmModel.FilterConfig)
		for i := len(cfg) - 1; i >= 0; i-- {
			filter := cfg[i]
			ecdsName := key + "-" + filter.Name
			c, _ := filter.Config.(map[string]interface{})
			typeURL, _ := c["@type"].(string)
			ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches,
				&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
					ApplyTo: istioapi.EnvoyFilter_NETWORK_FILTER,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &istioapi.EnvoyFilter_ListenerMatch{
								Name: ldsName,
							},
						},
					},
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_INSERT_FIRST,
						Value: MustNewStruct(map[string]interface{}{
							"name": ecdsName,
							"config_discovery": map[string]interface{}{
								"config_source": map[string]interface{}{
									"ads": map[string]interface{}{},
								},
								"type_urls": []interface{}{
									typeURL,
								},
							},
						}),
					},
				},
				&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
					ApplyTo: istioapi.EnvoyFilter_EXTENSION_CONFIG,
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_ADD,
						Value: MustNewStruct(map[string]interface{}{
							"name":         ecdsName,
							"typed_config": filter.Config,
						}),
					},
				},
			)
		}
	}

	if hasHCM {
		cfg := config[model.CategoryECDSGolang]
		if cfg == nil {
			cfg = map[string]interface{}{}
		}
		ecdsName := key + "-" + model.CategoryGolangPlugins
		ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches,
			&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: istioapi.EnvoyFilter_HTTP_FILTER,
				Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
					ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
						Listener: &istioapi.EnvoyFilter_ListenerMatch{
							Name: ldsName,
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
					Value: MustNewStruct(map[string]interface{}{
						"name": ecdsName,
						"config_discovery": map[string]interface{}{
							"apply_default_config_without_warming": true,
							"default_config": map[string]interface{}{
								"@type":        "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
								"library_id":   "fm",
								"library_path": ctrlcfg.GoSoPath(),
								"plugin_name":  "fm",
							},
							"config_source": map[string]interface{}{
								"ads": map[string]interface{}{},
							},
							"type_urls": []interface{}{
								"type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
							},
						},
					}),
				},
			},
			&istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: istioapi.EnvoyFilter_EXTENSION_CONFIG,
				Patch: &istioapi.EnvoyFilter_Patch{
					Operation: istioapi.EnvoyFilter_Patch_ADD,
					Value: MustNewStruct(map[string]interface{}{
						"name": ecdsName,
						"typed_config": map[string]interface{}{
							"@type":        "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
							"library_id":   "fm",
							"library_path": ctrlcfg.GoSoPath(),
							"plugin_name":  "fm",
							"plugin_config": map[string]interface{}{
								"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
								"value": cfg,
							},
						},
					}),
				},
			},
		)
	}

	return ef
}

func GenerateConsumers(consumers map[string]interface{}) *istiov1a3.EnvoyFilter {
	return &istiov1a3.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ctrlcfg.RootNamespace(),
			Name:      ECDSConsumerName,
			Labels: map[string]string{
				constant.LabelCreatedBy: "Consumer",
			},
		},
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
								"library_path": ctrlcfg.GoSoPath(),
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

func GenerateDynamicConfigs(namespacedDynamicConfigs map[string]map[string]*mosniov1.DynamicConfig) map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter {
	efs := map[component.EnvoyFilterKey]*istiov1a3.EnvoyFilter{}
	for ns, dynamicConfigs := range namespacedDynamicConfigs {
		ef := &istiov1a3.EnvoyFilter{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      DynamicConfigEnvoyFilterName,
				Labels: map[string]string{
					constant.LabelCreatedBy: "DynamicConfig",
				},
			},
			Spec: istioapi.EnvoyFilter{
				ConfigPatches: []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{},
			},
		}
		// Each DynamicConfig is smaller than 1.5MB, which is the limit applied by the k8s API server (the value may be different by configured).
		// In prod, we generate the EnvoyFilter inside the istio, so the size of EnvoyFilter doesn't matter.

		configs := make([]*mosniov1.DynamicConfig, 0, len(dynamicConfigs))
		for _, dynamicConfig := range dynamicConfigs {
			configs = append(configs, dynamicConfig)
		}
		sort.Slice(configs, func(i, j int) bool {
			return configs[i].Spec.Type < configs[j].Spec.Type
		})

		httpFilters := []interface{}{}
		for _, cfg := range configs {
			var dispatchedConfig interface{}
			_ = json.Unmarshal(cfg.Spec.Config.Raw, &dispatchedConfig)

			ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, &istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: istioapi.EnvoyFilter_EXTENSION_CONFIG,
				Patch: &istioapi.EnvoyFilter_Patch{
					Operation: istioapi.EnvoyFilter_Patch_ADD,
					Value: MustNewStruct(map[string]interface{}{
						"name": fmt.Sprintf("htnn-DynamicConfig-%s", cfg.Spec.Type),
						"typed_config": map[string]interface{}{
							"@type":        "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
							"library_id":   "dc",
							"library_path": ctrlcfg.GoSoPath(),
							"plugin_name":  "dc",
							"plugin_config": map[string]interface{}{
								"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
								"value": map[string]interface{}{
									"name":   cfg.Spec.Type,
									"config": dispatchedConfig,
								},
							},
						},
					}),
				},
			})
			httpFilters = append(httpFilters, map[string]interface{}{
				"name": fmt.Sprintf("htnn-DynamicConfig-%s", cfg.Spec.Type),
				"config_discovery": map[string]interface{}{
					"config_source": map[string]interface{}{
						"ads": map[string]interface{}{},
					},
					"type_urls": []interface{}{
						"type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
					},
				},
			})
		}

		httpFilters = append(httpFilters, map[string]interface{}{
			"name": "envoy.filters.http.router",
			"typed_config": map[string]interface{}{
				"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
			},
		})
		listener := &istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
			ApplyTo: istioapi.EnvoyFilter_LISTENER,
			Patch: &istioapi.EnvoyFilter_Patch{
				Operation: istioapi.EnvoyFilter_Patch_ADD,
				Value: MustNewStruct(map[string]interface{}{
					"name":              "htnn_dynamic_config",
					"internal_listener": map[string]interface{}{},
					"filter_chains": []interface{}{
						map[string]interface{}{
							"filters": []interface{}{
								map[string]interface{}{
									"name": "envoy.filters.network.http_connection_manager",
									"typed_config": map[string]interface{}{
										"@type":        "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
										"stat_prefix":  "htnn_dynamic_config",
										"http_filters": httpFilters,
										"route_config": map[string]interface{}{
											"name": "htnn_dynamic_config",
											"virtual_hosts": []interface{}{
												map[string]interface{}{
													"name":    "htnn_dynamic_config",
													"domains": []interface{}{"*"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				),
			},
		}
		ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, listener)
		efs[component.EnvoyFilterKey{
			Namespace: ns,
			Name:      ef.Name,
		}] = ef
	}

	return efs
}
