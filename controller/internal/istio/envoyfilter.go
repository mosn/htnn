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

func GenerateHostFilter(host *model.VirtualHost, config *filtermanager.FilterManagerConfig) *istiov1a3.EnvoyFilter {
	v := map[string]interface{}{}
	// This Marshal/Unmarshal trick works around the type check in MustNewStruct
	data, _ := json.Marshal(config)
	json.Unmarshal(data, &v)
	return &istiov1a3.EnvoyFilter{
		Spec: istioapi.EnvoyFilter{
			ConfigPatches: []*istioapi.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: istioapi.EnvoyFilter_VIRTUAL_HOST,
					Match: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch{
						ObjectTypes: &istioapi.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{
							RouteConfiguration: &istioapi.EnvoyFilter_RouteConfigurationMatch{
								Vhost: &istioapi.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{
									Name: host.Name,
								},
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
