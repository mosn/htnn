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

	"google.golang.org/protobuf/types/known/structpb"
	istioapi "istio.io/api/networking/v1alpha3"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlcfg "mosn.io/moe/controller/internal/config"
	"mosn.io/moe/controller/internal/model"
	"mosn.io/moe/pkg/filtermanager"
)

const (
	DefaultHttpFilter = "htnn-http-filter"
)

func MustNewStruct(fields map[string]interface{}) *structpb.Struct {
	st, err := structpb.NewStruct(fields)
	if err != nil {
		// NewStruct returns error only when the fields contain non-standard type
		panic(err)
	}
	return st
}

func DefaultEnvoyFilters() map[string]*istiov1a3.EnvoyFilter {
	efs := map[string]*istiov1a3.EnvoyFilter{}
	efs[DefaultHttpFilter] = &istiov1a3.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultHttpFilter,
		},
		Spec: istioapi.EnvoyFilter{
			ConfigPatches: []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: istioapi.EnvoyFilter_HTTP_FILTER,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						// As currently we only support Gateway cases, here we hardcode the context
						// to default envoy filters. We don't need to do that for user-defined envoy
						// filters. Because adding that will require a break change to remove it.
						// And user-defined envoy filter won't apply to mesh because:
						// 1. We don't support attaching policy to mesh.
						// 2. Mesh configuration doesn't have Go HTTP filter.
						Context: istioapi.EnvoyFilter_GATEWAY,
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
					Patch: &istioapi.EnvoyFilter_Patch{
						Operation: istioapi.EnvoyFilter_Patch_INSERT_BEFORE,
						Value: MustNewStruct(map[string]interface{}{
							"name": "envoy.filters.http.golang",
							"typed_config": map[string]interface{}{
								"@type":        "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.Config",
								"library_id":   "fm",
								"library_path": ctrlcfg.GoSoPath(),
								"plugin_name":  "fm",
							},
						}),
					},
				},
			},
		},
	}
	return efs
}

func GenerateRouteFilter(host *model.VirtualHost, route string, config *filtermanager.FilterManagerConfig) *istiov1a3.EnvoyFilter {
	v := map[string]interface{}{}
	// This Marshal/Unmarshal trick works around the type check in MustNewStruct
	data, _ := json.Marshal(config)
	_ = json.Unmarshal(data, &v)

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
							"typed_per_filter_config": map[string]interface{}{
								"envoy.filters.http.golang": map[string]interface{}{
									"@type": "type.googleapis.com/envoy.extensions.filters.http.golang.v3alpha.ConfigsPerRoute",
									"plugins_config": map[string]interface{}{
										"fm": map[string]interface{}{
											"config": map[string]interface{}{
												"@type": "type.googleapis.com/xds.type.v3.TypedStruct",
												"value": v,
											},
										},
									},
								},
							},
						}),
					},
				},
			},
		},
	}
}
