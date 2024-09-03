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

package pkg

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	mosniov1 "mosn.io/htnn/types/apis/v1"
)

// MapToObj convert the Go struct to k8s client.Object
func MapToObj(in map[string]interface{}) client.Object {
	var out client.Object
	data, _ := json.Marshal(in)
	group := in["apiVersion"].(string)
	if strings.HasPrefix(group, "networking.istio.io") {
		switch in["kind"] {
		case "VirtualService":
			out = &istiov1a3.VirtualService{}
		case "Gateway":
			out = &istiov1a3.Gateway{}
		case "EnvoyFilter":
			out = &istiov1a3.EnvoyFilter{}
		}
	} else if strings.HasPrefix(group, "gateway.networking.k8s.io") {
		switch in["kind"] {
		case "HTTPRoute":
			out = &gwapiv1b1.HTTPRoute{}
		case "Gateway":
			out = &gwapiv1b1.Gateway{}
		}
	} else if strings.HasPrefix(group, "htnn.mosn.io") {
		switch in["kind"] {
		case "FilterPolicy":
			out = &mosniov1.FilterPolicy{}
		case "Consumer":
			out = &mosniov1.Consumer{}
		case "ServiceRegistry":
			out = &mosniov1.ServiceRegistry{}
		case "DynamicConfig":
			out = &mosniov1.DynamicConfig{}
		}
	}
	if out == nil {
		panic("unknown crd")
	}
	json.Unmarshal(data, out)
	return out
}

// FakeK8sClient returns a fake k8s client that can be mocked for testing
func FakeK8sClient(t *testing.T) client.Client {
	cfg := &rest.Config{}
	k8sClient, err := client.New(cfg, client.Options{})
	require.NoError(t, err)
	return k8sClient
}

func DeleteK8sResource(ctx context.Context, k8sClient client.Client, obj client.Object, opts ...client.DeleteOption) {
	err := k8sClient.Delete(ctx, obj, opts...)
	if err != nil && !apierrors.IsNotFound(err) {
		Expect(err).Should(Succeed())
	}
}
