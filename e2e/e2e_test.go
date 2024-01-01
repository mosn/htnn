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

package e2e

import (
	"testing"

	istioscheme "istio.io/client-go/pkg/clientset/versioned/scheme"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	mosniov1 "mosn.io/htnn/controller/api/v1"
	"mosn.io/htnn/e2e/pkg/suite"
	_ "mosn.io/htnn/e2e/tests" // import all tests
)

func TestE2E(t *testing.T) {
	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("Error loading Kubernetes config: %v", err)
	}
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("Error initializing Kubernetes client: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Error initializing Kubernetes REST client: %v", err)
	}

	err = istioscheme.AddToScheme(client.Scheme())
	if err != nil {
		t.Fatalf("Error adding Istio types to scheme: %v", err)
	}

	err = gwapiv1.AddToScheme(client.Scheme())
	if err != nil {
		t.Fatalf("Error adding k8s gateway API types to scheme: %v", err)
	}

	err = mosniov1.AddToScheme(client.Scheme())
	if err != nil {
		t.Fatalf("Error adding mosn types to scheme: %v", err)
	}

	st := suite.New(suite.Options{
		Client:    client,
		Clientset: clientset,
	})
	st.Run(t)
}
