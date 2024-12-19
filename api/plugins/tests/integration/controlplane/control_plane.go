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

package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	xds "github.com/cncf/xds/go/xds/type/v3"
	golang "github.com/envoyproxy/go-control-plane/contrib/envoy/extensions/filters/http/golang/v3alpha"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/htnn/api/internal/proto"
	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/pkg/log"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
)

var (
	logger = log.DefaultLogger.WithName("controlplane")
)

type ControlPlane struct {
	version       int
	snapshotCache cache.SnapshotCache
	grpcServer    *grpc.Server
}

func NewControlPlane() *ControlPlane {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := server.NewServer(context.Background(), snapshotCache, nil)
	grpcServer := grpc.NewServer()
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	cp := &ControlPlane{
		snapshotCache: snapshotCache,
		grpcServer:    grpcServer,
	}
	return cp
}

func getRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

func isWsl() bool {
	f, err := os.Open("/proc/version")
	if err != nil {
		return false
	}
	d, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	return strings.Contains(string(d), "-WSL")
}

func (cp *ControlPlane) Start() {
	host := "127.0.0.1"
	if runtime.GOOS == "linux" && !isWsl() {
		// We need to use 0.0.0.0 on Linux so the data plane in the Docker
		// can connect to it.
		host = "0.0.0.0"
		// Use 0.0.0.0 on Mac will be prompted by some security policies so we
		// only use it on Linux.
	}

	port := ":9999"
	portEnv := os.Getenv("TEST_ENVOY_CONTROL_PLANE_PORT")
	if portEnv != "" {
		port = ":" + portEnv
	}

	lis, err := net.Listen("tcp", host+port)
	if err != nil {
		logger.Error(err, "failed to listen")
		return
	}

	if err := cp.grpcServer.Serve(lis); err != nil {
		logger.Error(err, "failed to start control plane")
	}
}

type Resources map[resource.Type][]types.Resource

func (cp *ControlPlane) updateConfig(t *testing.T, res Resources) {
	snapshot, err := cache.NewSnapshot(fmt.Sprintf("%v.0", cp.version), res)
	if err != nil {
		logger.Error(err, "failed to new snapshot")
		return
	}

	cp.version++
	var IDs []string
	// wait for DP to connect CP
	require.Eventually(t, func() bool {
		IDs = cp.snapshotCache.GetStatusKeys()
		return len(IDs) > 0
	}, 10*time.Second, 10*time.Millisecond, "failed to wait for DP to connect CP")

	for _, id := range IDs {
		err = cp.snapshotCache.SetSnapshot(context.Background(), id, snapshot)
		require.Nil(t, err, "failed to set snapshot")
	}
}

func (cp *ControlPlane) UseGoPluginConfig(t *testing.T, config *filtermanager.FilterManagerConfig, dp *dataplane.DataPlane) {
	testRoute := &route.Route{
		Match: &route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: "backend",
				},
			},
		},
	}
	if config != nil {
		pluginName := os.Getenv("plugin_name_for_test")
		if pluginName == "" {
			pluginName = "fm"
		}
		testRoute.TypedPerFilterConfig = map[string]*any1.Any{
			"htnn.filters.http.golang": proto.MessageToAny(&golang.ConfigsPerRoute{
				PluginsConfig: map[string]*golang.RouterPlugin{
					pluginName: {
						Override: &golang.RouterPlugin_Config{
							Config: proto.MessageToAny(
								FilterManagerConfigToTypedStruct(config)),
						},
					},
				},
			}),
		}
	}

	cp.updateConfig(t, Resources{
		resource.RouteType: []types.Resource{
			&route.RouteConfiguration{
				Name: "dynamic_route",
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "dynmamic_service",
						Domains: []string{"*"},
						Routes: []*route.Route{
							{
								Name: getRandomString(8),
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Path{
										Path: "/detect_if_the_rds_takes_effect",
									},
								},
								Action: &route.Route_DirectResponse{
									DirectResponse: &route.DirectResponseAction{
										Status: 200,
									},
								},
								TypedPerFilterConfig: map[string]*any1.Any{
									"htnn.filters.http.golang": proto.MessageToAny(&golang.ConfigsPerRoute{
										PluginsConfig: map[string]*golang.RouterPlugin{
											"fm": {
												Override: &golang.RouterPlugin_Config{
													Config: proto.MessageToAny(
														FilterManagerConfigToTypedStruct(NewSinglePluginConfig("detector", nil))),
												},
											},
										},
									}),
								},
							},
							{
								Match: &route.RouteMatch{
									PathSpecifier: &route.RouteMatch_Path{
										Path: "/flush_coverage",
									},
								},
								Action: &route.Route_DirectResponse{
									DirectResponse: &route.DirectResponseAction{
										Status: 200,
									},
								},
								TypedPerFilterConfig: map[string]*any1.Any{
									"htnn.filters.http.golang": proto.MessageToAny(&golang.ConfigsPerRoute{
										PluginsConfig: map[string]*golang.RouterPlugin{
											"fm": {
												Override: &golang.RouterPlugin_Config{
													Config: proto.MessageToAny(
														FilterManagerConfigToTypedStruct(NewSinglePluginConfig("coverage", nil))),
												},
											},
										},
									}),
								},
							},
							testRoute,
						},
					},
				},
			},
		},
	})

	// Wait for DP to use the configuration.
	require.Eventually(t, func() bool {
		return dp.Configured()
	}, 10*time.Second, 50*time.Millisecond, "failed to wait for DP to use the configuration")
}

func FilterManagerConfigToTypedStruct(fmc *filtermanager.FilterManagerConfig) *xds.TypedStruct {
	v := map[string]interface{}{}
	data, _ := json.Marshal(fmc)
	json.Unmarshal(data, &v)
	st, err := structpb.NewStruct(v)
	if err != nil {
		logger.Error(err, "failed to TypedStruct", "FilterManagerConfig", fmc)
		return nil
	}
	return &xds.TypedStruct{
		Value: st,
	}
}

func NewSinglePluginConfig(name string, config interface{}) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{}
	fmc.Namespace = "ns"
	fmc.Plugins = []*model.FilterConfig{{Name: name, Config: config}}
	return fmc
}

func NewPluginConfig(plugins []*model.FilterConfig) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{}
	fmc.Namespace = "ns"
	fmc.Plugins = plugins
	return fmc
}
