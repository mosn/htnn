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

package output

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	envoycfgcorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	mcpapi "istio.io/api/mcp/v1alpha1"

	"mosn.io/htnn/controller/internal/config"
)

func MarshalToMcpPb(name string, src proto.Message) (*anypb.Any, error) {
	body := &anypb.Any{}
	if err := anypb.MarshalFrom(body, src, proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("failed to marshal mcp body: %w", err)
	}

	ns := config.RootNamespace()
	mcpRes := &mcpapi.Resource{
		Metadata: &mcpapi.Metadata{
			Name: fmt.Sprintf("%s/%s", ns, name),
		},
		Body: body,
	}

	pb := &anypb.Any{}
	if err := anypb.MarshalFrom(pb, mcpRes, proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("failed to marshal mcp resource: %w", err)
	}

	return pb, nil
}

type (
	DiscoveryStream      = discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer
	DeltaDiscoveryStream = discovery.AggregatedDiscoveryService_DeltaAggregatedResourcesServer
)

type mcpServer struct {
	logger *logr.Logger

	subscribers      sync.Map
	nextSubscriberID atomic.Uint64

	resourceLock   sync.RWMutex
	envoyFilters   []*anypb.Any
	serviceEntries []*anypb.Any
}

func NewMcpServer(logger *logr.Logger) *mcpServer {
	return &mcpServer{
		logger: logger,
	}
}

type subscriber struct {
	id uint64

	stream      DiscoveryStream
	closeStream func()
}

func (srv *mcpServer) UpdateEnvoyFilters(envoyFilters []*anypb.Any) {
	srv.resourceLock.Lock()
	srv.envoyFilters = envoyFilters
	srv.resourceLock.Unlock()
	go func() {
		typeUrl := "networking.istio.io/v1alpha3/EnvoyFilter"
		srv.sendToSubscribers(typeUrl, envoyFilters)
	}()
}

func (srv *mcpServer) UpdateServiceEntries(serviceEntries []*anypb.Any) {
	srv.resourceLock.Lock()
	srv.serviceEntries = serviceEntries
	srv.resourceLock.Unlock()
	go func() {
		typeUrl := "networking.istio.io/v1beta1/ServiceEntry"
		srv.sendToSubscribers(typeUrl, serviceEntries)
	}()
}

func (srv *mcpServer) send(sub *subscriber, typeUrl string, mcpResources []*anypb.Any) {
	if err := sub.stream.Send(&discovery.DiscoveryResponse{
		TypeUrl:     typeUrl,
		VersionInfo: strconv.FormatInt(time.Now().UnixNano(), 10),
		Resources:   mcpResources,
		ControlPlane: &envoycfgcorev3.ControlPlane{
			Identifier: os.Getenv("POD_NAME"),
		},
	}); err != nil {
		id := sub.id
		srv.logger.Error(err, "failed to send to subscriber", "id", id)
		// let Istio to retry
		sub.closeStream()
		srv.subscribers.Delete(id)
	}
}

func (srv *mcpServer) sendToSubscribers(typeUrl string, mcpResources []*anypb.Any) {
	srv.resourceLock.Lock()
	defer srv.resourceLock.Unlock()

	srv.subscribers.Range(func(key, value any) bool {
		srv.logger.Info("sending to subscriber", "id", key, "typeUrl", typeUrl, "length", len(mcpResources))
		srv.send(value.(*subscriber), typeUrl, mcpResources)
		return true
	})
}

func (srv *mcpServer) CloseSubscribers() {
	srv.subscribers.Range(func(key, value any) bool {
		srv.logger.Info("close subscriber", "id", key)
		value.(*subscriber).closeStream()
		srv.subscribers.Delete(key)

		return true
	})
}

func (srv *mcpServer) initSubscriberResource(sub *subscriber) {
	srv.resourceLock.Lock()
	defer srv.resourceLock.Unlock()

	srv.logger.Info("sending initial conf to subscriber", "id", sub.id)
	typeUrl := "networking.istio.io/v1beta1/ServiceEntry"
	srv.send(sub, typeUrl, srv.serviceEntries)
	typeUrl = "networking.istio.io/v1alpha3/EnvoyFilter"
	srv.send(sub, typeUrl, srv.envoyFilters)
}

// Implement discovery.AggregatedDiscoveryServiceServer

func (srv *mcpServer) StreamAggregatedResources(downstream DiscoveryStream) error {
	ctx, closeStream := context.WithCancel(downstream.Context())

	sub := &subscriber{
		id:          srv.nextSubscriberID.Add(1),
		stream:      downstream,
		closeStream: closeStream,
	}
	srv.logger.Info("handle new subscriber", "id", sub.id)

	srv.subscribers.Store(sub.id, sub)

	go func() {
		srv.initSubscriberResource(sub)
	}()

	<-ctx.Done()
	return nil
}

func (srv *mcpServer) DeltaAggregatedResources(downstream DeltaDiscoveryStream) error {
	// By now, Istio doesn't support MCP over delta ads
	return status.Errorf(codes.Unimplemented, "not implemented")
}

var _ discovery.AggregatedDiscoveryServiceServer = (*mcpServer)(nil)
