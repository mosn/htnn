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

package control_plane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"mosn.io/htnn/internal/proto"
	"mosn.io/htnn/pkg/filtermanager"
	"mosn.io/htnn/pkg/filtermanager/model"
	"mosn.io/htnn/pkg/log"
	"mosn.io/htnn/plugins/tests/integration/data_plane"
)

var (
	logger = log.DefaultLogger.WithName("control_plane")
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

	lis, _ := net.Listen("tcp", host+":9999")
	if err := cp.grpcServer.Serve(lis); err != nil {
		logger.Error(err, "failed to start control plane")
	}
}

func eventually(waitFor time.Duration, tick time.Duration, condition func() bool) error {
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case <-timer.C:
			return errors.New("Condition never satisfied")
		case <-tick:
			tick = nil
			go func() { ch <- condition() }()
		case v := <-ch:
			if v {
				return nil
			}
			tick = ticker.C
		}
	}
}

type Resources map[resource.Type][]types.Resource

func (cp *ControlPlane) updateConfig(res Resources) {
	snapshot, err := cache.NewSnapshot(fmt.Sprintf("%v.0", cp.version), res)
	if err != nil {
		logger.Error(err, "failed to new snapshot")
		return
	}

	cp.version++
	var IDs []string
	// wait for DP to connect CP
	err = eventually(10*time.Second, 10*time.Millisecond, func() bool {
		IDs = cp.snapshotCache.GetStatusKeys()
		return len(IDs) > 0
	})
	if err != nil {
		logger.Error(err, "failed to wait for DP to connect CP")
		return
	}

	for _, id := range IDs {
		err = cp.snapshotCache.SetSnapshot(context.Background(), id, snapshot)
		if err != nil {
			logger.Error(err, "failed to set snapshot")
			return
		}
	}
}

func (cp *ControlPlane) UseGoPluginConfig(config *filtermanager.FilterManagerConfig, dp *data_plane.DataPlane) {
	cp.updateConfig(Resources{
		resource.RouteType: []types.Resource{
			&route.RouteConfiguration{
				Name: "dynamic_route",
				VirtualHosts: []*route.VirtualHost{
					{
						Name:    "dynmamic_service",
						Domains: []string{"*"},
						Routes: []*route.Route{
							{
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
														FilterManagerConfigToTypedStruct(NewSinglePluinConfig("coverage", nil))),
												},
											},
										},
									}),
								},
							},
							{
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
								TypedPerFilterConfig: map[string]*any1.Any{
									"htnn.filters.http.golang": proto.MessageToAny(&golang.ConfigsPerRoute{
										PluginsConfig: map[string]*golang.RouterPlugin{
											"fm": {
												Override: &golang.RouterPlugin_Config{
													Config: proto.MessageToAny(
														FilterManagerConfigToTypedStruct(config)),
												},
											},
										},
									}),
								},
							},
						},
					},
				},
			},
		},
	})

	// Wait for DP to use the configuration. Unlike the assert.Eventually, this function doesn't
	// fail the test when it timed out.
	eventually(5*time.Second, 50*time.Millisecond, func() bool {
		return dp.Configured()
	})
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

func NewSinglePluinConfig(name string, config interface{}) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{}
	fmc.Namespace = "ns"
	fmc.Plugins = []*model.FilterConfig{{Name: name, Config: config}}
	return fmc
}

func NewPluinConfig(plugins []*model.FilterConfig) *filtermanager.FilterManagerConfig {
	fmc := &filtermanager.FilterManagerConfig{}
	fmc.Namespace = "ns"
	fmc.Plugins = plugins
	return fmc
}
