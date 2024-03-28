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

package registry

import (
	"fmt"
	"strings"

	istioapi "istio.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"mosn.io/htnn/controller/internal/log"
	"mosn.io/htnn/types/pkg/registry"
)

// The protocol defined here should match the protocol field in istio's ServicePort
// See https://github.com/istio/api/issues/3056
type Protocol string

const (
	HTTP        Protocol = "HTTP"
	HTTPS       Protocol = "HTTPS"
	GRPC        Protocol = "GRPC"
	HTTP2       Protocol = "HTTP2"
	MONGO       Protocol = "MONGO"
	TCP         Protocol = "TCP"
	TLS         Protocol = "TLS"
	Unsupported Protocol = "Unsupported"
)

var ProtocolMap = map[string]Protocol{
	"http":  HTTP,
	"https": HTTPS,
	"grpc":  GRPC,
	"http2": HTTP2,
	"mongo": MONGO,
	"tcp":   TCP,
	"tls":   TLS,
}

func ParseProtocol(s string) Protocol {
	res, ok := ProtocolMap[strings.ToLower(s)]
	if !ok {
		return Unsupported
	}
	return res
}

// ServiceEntryWrapper is a wrapper around the istio's ServiceEntry
type ServiceEntryWrapper struct {
	istioapi.ServiceEntry
	Source string
}

// ServiceEntryStore is the store of ServiceEntryWrapper. The service must be a valid k8s service name.
// It will be used as both the name of the ServiceEntry used by Istio (the unique key in control plane),
// and the domain of the cluster used by Envoy (the unique key in data plane).
type ServiceEntryStore interface {
	Update(service string, se *ServiceEntryWrapper)
	Delete(service string)
}

// Registry is the interface that all registries must implement
type Registry interface {
	registry.Registry

	Start(config registry.RegistryConfig) error
	Stop() error
	// Reload provides an effective way to update the configuration than Start & Stop
	Reload(config registry.RegistryConfig) error
}

// RegistryFactory provides methods to prepare configuration & create registry
type RegistryFactory func(store ServiceEntryStore, om metav1.ObjectMeta) (Registry, error)

var (
	registryFactories = make(map[string]RegistryFactory)
)

// AddRegistryFactory will be used by the user to register a new registry
func AddRegistryFactory(name string, factory RegistryFactory) {
	log.Infof("register registry %s", name)

	// override plugin is allowed so that we can patch plugin with bugfix if upgrading
	// the whole htnn is not available
	registryFactories[name] = factory
}

// CreateRegistry is called by HTNN to create a new registry
func CreateRegistry(name string, store ServiceEntryStore, om metav1.ObjectMeta) (Registry, error) {
	factory, ok := registryFactories[name]
	if !ok {
		return nil, fmt.Errorf("unknown registry %s", name)
	}

	return factory(store, om)
}
