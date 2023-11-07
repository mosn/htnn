package control_plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"runtime"
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

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/pkg/log"
	"mosn.io/moe/pkg/proto"
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

func (cp *ControlPlane) Start() {
	host := "127.0.0.1"
	if runtime.GOOS == "linux" {
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

type Resources map[resource.Type][]types.Resource

func (cp *ControlPlane) updateConfig(res Resources) {
	snapshot, err := cache.NewSnapshot(fmt.Sprintf("%v.0", cp.version), res)
	if err != nil {
		logger.Error(err, "failed to new snapshot")
		return
	}

	cp.version++
	IDs := cp.snapshotCache.GetStatusKeys()
	for _, id := range IDs {
		logger.Info("dispatch config", "snapshot", snapshot, "id", id)
		err = cp.snapshotCache.SetSnapshot(context.Background(), id, snapshot)
		if err != nil {
			logger.Error(err, "failed to set snapshot")
			return
		}
	}

	// wait for DP to use the configuration
	time.Sleep(1 * time.Second)
}

func (cp *ControlPlane) UseGoPluginConfig(config *filtermanager.FilterManagerConfig) {
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
									"envoy.filters.http.golang": proto.MessageToAny(&golang.ConfigsPerRoute{
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
	fmc.Plugins = []*filtermanager.FilterConfig{{Name: name, Config: config}}
	return fmc
}
