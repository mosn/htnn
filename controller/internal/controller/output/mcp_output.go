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
	"net"
	"sync"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/pkg/procession"
	"mosn.io/htnn/pkg/log"
)

type mcpOutput struct {
	logger *logr.Logger

	// envoyFilters will be updated by multiple sources
	envoyFilters sync.Map

	mcp *mcpServer
}

func NewMcpOutput(ctx context.Context) (procession.Output, error) {
	logger := log.DefaultLogger.WithName("mcp output")
	s := grpc.NewServer()

	srv := NewMcpServer(&logger)

	discovery.RegisterAggregatedDiscoveryServiceServer(s, srv)

	addr := config.McpServerListenAddress()
	logger.Info("listening as mcp server", "address", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		ctx, cancel := context.WithCancel(ctx)

		go func() {
			defer cancel()

			logger.Info("starting mcp server")

			err := s.Serve(l)
			if err != nil {
				logger.Error(err, "mcp server failed")
			}
		}()

		<-ctx.Done()
		logger.Info("stopping mcp server")
		srv.CloseSubscribers()
		s.GracefulStop()
		logger.Info("mcp server stopped")
	}()

	return &mcpOutput{
		logger: &logger,
		mcp:    srv,
	}, nil
}

func (o *mcpOutput) WriteEnvoyFilters(ctx context.Context, src procession.ConfigSource, filters map[string]*istiov1a3.EnvoyFilter) error {
	// Store the converted Any directly can save memory, but we keep the original EnvoyFilter here
	// so that we can add observability in the future.
	o.envoyFilters.Store(src, filters)
	ress := make([]*anypb.Any, 0, len(filters)*2)
	ok := true
	o.envoyFilters.Range(func(_, value interface{}) bool {
		efs := value.(map[string]*istiov1a3.EnvoyFilter)
		for name, ef := range efs {
			res, err := MarshalToMcpPb(name, &ef.Spec)
			if err != nil {
				o.logger.Error(err, "failed to marshal EnvoyFilter", "name", name)
				// do not push partial configuration, this may cause service unavailable
				ok = false
				return false
			}
			ress = append(ress, res)
		}
		return true
	})

	if ok {
		o.mcp.UpdateEnvoyFilters(ress)
	}
	return nil
}

func (o *mcpOutput) WriteServiceEntries(ctx context.Context, src procession.ConfigSource, serviceEntries map[string]*istioapi.ServiceEntry) {
	ress := make([]*anypb.Any, 0, len(serviceEntries))
	for name, se := range serviceEntries {
		res, err := MarshalToMcpPb(name, se)
		if err != nil {
			o.logger.Error(err, "failed to marshal ServiceEntry", "name", name)
			// do not push partial configuration, this may cause service unavailable
			return
		}
		ress = append(ress, res)
	}

	o.mcp.UpdateServiceEntries(ress)
}
