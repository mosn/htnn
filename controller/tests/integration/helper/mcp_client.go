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

package helper

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	mcpapi "istio.io/api/mcp/v1alpha1"
	istioapi "istio.io/api/networking/v1beta1"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"mosn.io/htnn/controller/internal/config"
	"mosn.io/htnn/controller/internal/controller/output"
	"mosn.io/htnn/controller/internal/model"
	"mosn.io/htnn/controller/pkg/procession"
)

type mcpClient struct {
	// Stream is the GRPC connection stream, allowing direct GRPC send operations.
	// Set after Dial is called.
	stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesClient
	// xds client used to create a stream
	client discovery.AggregatedDiscoveryServiceClient
	conn   *grpc.ClientConn

	lock sync.Mutex

	k8sClient client.Client
	// To simulate k8s output in the existing tests, the simplest way is to use the k8s output directly
	output procession.Output
}

func NewMcpClient(cli client.Client) *mcpClient {
	c := &mcpClient{
		k8sClient: cli,
		output:    output.NewK8sOutput(cli),
	}
	return c
}

func (c *mcpClient) dial() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, config.McpServerListenAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // ensure the connection is established
	)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *mcpClient) Init() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for i := 0; i < 10; i++ {
		err := c.dial()
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	c.client = discovery.NewAggregatedDiscoveryServiceClient(c.conn)
	var err error
	c.stream, err = c.client.StreamAggregatedResources(context.Background())
	Expect(err).NotTo(HaveOccurred())

	// For now we don't care about the details in DiscoveryRequest
	req := &discovery.DiscoveryRequest{}
	err = c.stream.Send(req)
	Expect(err).NotTo(HaveOccurred())
}

const (
	TypeUrlEnvoyFilter  = "networking.istio.io/v1alpha3/EnvoyFilter"
	TypeUrlServiceEntry = "networking.istio.io/v1beta1/ServiceEntry"
)

func (c *mcpClient) Handle() {
	for {
		var err error
		msg, err := c.stream.Recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				Expect(err).NotTo(HaveOccurred())
			}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		switch msg.TypeUrl {
		case TypeUrlEnvoyFilter:
			efs := map[string]*istiov1a3.EnvoyFilter{}
			for _, resource := range msg.Resources {
				ef := c.convertAnyToEnvoyFilter(resource)
				efs[ef.Name] = ef
			}
			if _, ok := efs[model.ConsumerEnvoyFilterName]; ok {
				c.writeEnvoyFiltersWithRetry(ctx, false, efs)
				delete(efs, model.ConsumerEnvoyFilterName)
			} else {
				// all EnvoyFilters don't contain consumer EnvoyFilter, remove it from k8s
				var ef istiov1a3.EnvoyFilter
				ns := config.RootNamespace()
				err := c.k8sClient.Get(ctx, types.NamespacedName{Name: model.ConsumerEnvoyFilterName, Namespace: ns}, &ef)
				if err == nil {
					err = c.k8sClient.Delete(ctx, &ef)
					Expect(err).NotTo(HaveOccurred())
				}
			}
			// handle EnvoyFilters except the one from consumer
			c.writeEnvoyFiltersWithRetry(ctx, true, efs)
		case TypeUrlServiceEntry:
			ses := map[string]*istioapi.ServiceEntry{}
			for _, resource := range msg.Resources {
				se := c.convertAnyToServiceEntry(resource)
				ses[se.Name] = &se.Spec
			}
			c.output.FromServiceRegistry(ctx, ses)
		default:
			Expect(false).To(BeTrue(), "unknown type url: %s", msg.TypeUrl)
		}
	}
}

func (c *mcpClient) convertAnyToEnvoyFilter(res *anypb.Any) *istiov1a3.EnvoyFilter {
	mcpRes := &mcpapi.Resource{}
	err := res.UnmarshalTo(mcpRes)
	Expect(err).NotTo(HaveOccurred())

	ef := &istiov1a3.EnvoyFilter{}
	ss := strings.Split(mcpRes.Metadata.Name, "/")
	ef.SetNamespace(ss[0])
	ef.SetName(ss[1])
	ef.SetAnnotations(mcpRes.Metadata.Annotations)
	ef.SetLabels(mcpRes.Metadata.Labels)
	err = mcpRes.Body.UnmarshalTo(&ef.Spec)
	Expect(err).NotTo(HaveOccurred())
	return ef
}

func (c *mcpClient) convertAnyToServiceEntry(res *anypb.Any) *istiov1b1.ServiceEntry {
	mcpRes := &mcpapi.Resource{}
	err := res.UnmarshalTo(mcpRes)
	Expect(err).NotTo(HaveOccurred())

	se := &istiov1b1.ServiceEntry{}
	ss := strings.Split(mcpRes.Metadata.Name, "/")
	se.SetNamespace(ss[0])
	se.SetName(ss[1])
	err = mcpRes.Body.UnmarshalTo(&se.Spec)
	Expect(err).NotTo(HaveOccurred())
	return se
}

func (c *mcpClient) writeEnvoyFiltersWithRetry(ctx context.Context, fromHTTPFilterPolicy bool, filters map[string]*istiov1a3.EnvoyFilter) {
	err := retry.Do(
		func() error {
			// Here we simulate the reconcile when the write failed in k8s output
			// We deepcopy the filters as they are regenerated in the reconcile process
			efs := make(map[string]*istiov1a3.EnvoyFilter, len(filters))
			for name, ef := range filters {
				efs[name] = ef.DeepCopy()
			}
			if fromHTTPFilterPolicy {
				return c.output.FromHTTPFilterPolicy(ctx, efs)
			}
			return c.output.FromConsumer(ctx, efs[model.ConsumerEnvoyFilterName])
		},
		retry.RetryIf(func(err error) bool {
			return true
		}),
		retry.Attempts(3),
		// backoff delay
		retry.Delay(500*time.Millisecond),
	)
	Expect(err).NotTo(HaveOccurred())
}

func (c *mcpClient) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.conn == nil {
		return
	}
	c.conn.Close()
}
