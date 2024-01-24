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

package nacos

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	istioapi "istio.io/api/networking/v1beta1"

	"mosn.io/htnn/controller/pkg/registry"
)

func TestGenerateServiceEntry(t *testing.T) {
	host := "test.default-group.public.earth.nacos"
	reg := &Nacos{}

	type test struct {
		name     string
		services []model.SubscribeService
		port     *istioapi.ServicePort
		endpoint *istioapi.WorkloadEntry
	}
	tests := []test{}
	for input, proto := range registry.ProtocolMap {
		s := string(proto)
		tests = append(tests, test{
			name: input,
			services: []model.SubscribeService{
				{Port: 80, Ip: "1.1.1.1", Metadata: map[string]string{
					"protocol": input,
				}},
			},
			port: &istioapi.ServicePort{
				Name:     s,
				Protocol: s,
				Number:   80,
			},
			endpoint: &istioapi.WorkloadEntry{
				Address: "1.1.1.1",
				Ports:   map[string]uint32{s: 80},
				Labels: map[string]string{
					"protocol": input,
				},
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := reg.generateServiceEntry(host, tt.services)
			require.True(t, proto.Equal(se.ServiceEntry.Ports[0], tt.port))
			require.True(t, proto.Equal(se.ServiceEntry.Endpoints[0], tt.endpoint))
		})
	}
}
