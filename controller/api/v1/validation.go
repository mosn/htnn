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

package v1

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	istiov1b1 "istio.io/client-go/pkg/apis/networking/v1beta1"

	"mosn.io/htnn/controller/pkg/registry"
	"mosn.io/htnn/pkg/plugins"
)

func ValidateHTTPFilterPolicy(policy *HTTPFilterPolicy) error {
	ref := policy.Spec.TargetRef
	if ref.Namespace != nil {
		namespace := string(*ref.Namespace)
		if namespace != policy.Namespace {
			return errors.New("namespace in TargetRef doesn't match HTTPFilterPolicy's namespace")
		}
	}

	validTarget := false
	if ref.Group == "networking.istio.io" && ref.Kind == "VirtualService" {
		validTarget = true
	} else if ref.Group == "gateway.networking.k8s.io" && ref.Kind == "HTTPRoute" {
		validTarget = true
	}
	if !validTarget {
		return errors.New("unsupported targetRef.group or targetRef.kind")
	}

	for name, filter := range policy.Spec.Filters {
		p := plugins.LoadHttpPlugin(name)
		if p == nil {
			// reject unknown filter in CP, ignore unknown filter in DP
			return errors.New("unknown http filter: " + name)
		}

		data := filter.Config.Raw
		conf := p.Config()
		if err := protojson.Unmarshal(data, conf); err != nil {
			return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
		}

		if err := conf.Validate(); err != nil {
			return fmt.Errorf("invalid config for filter %s: %w", name, err)
		}
	}

	return nil
}

func ValidateVirtualService(vs *istiov1b1.VirtualService) error {
	if len(vs.Spec.Http) == 0 {
		return errors.New("only http route is supported")
	}
	for _, httpRoute := range vs.Spec.Http {
		if httpRoute.Name == "" {
			return errors.New("route name is empty")
		}
	}

	// TODO: support delegate VirtualService
	if len(vs.Spec.Hosts) == 0 {
		return errors.New("delegate VirtualService is not supported")
	}
	return nil
}

func ValidateGateway(gw *istiov1b1.Gateway) error {
	// TODO: support it
	for _, svr := range gw.Spec.Servers {
		for _, host := range svr.Hosts {
			if strings.ContainsRune(host, '/') {
				return errors.New("Gateway has host with namespace is not supported")
			}
		}
	}
	return nil
}

func ValidateConsumer(c *Consumer) error {
	for name, filter := range c.Spec.Auth {
		plugin := plugins.LoadHttpPlugin(name)
		if plugin == nil {
			// reject unknown filter in CP, ignore unknown filter in DP
			return errors.New("unknown authn filter: " + name)
		}
		p, ok := plugin.(plugins.ConsumerPlugin)
		if !ok {
			return errors.New("configured authn filter is not a consumer plugin: " + name)
		}

		data := filter.Config.Raw
		conf := p.ConsumerConfig()
		if err := protojson.Unmarshal(data, conf); err != nil {
			return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
		}

		if err := conf.Validate(); err != nil {
			return fmt.Errorf("invalid config for filter %s: %w", name, err)
		}
	}

	for name, filter := range c.Spec.Filters {
		p := plugins.LoadHttpPlugin(name)
		if p == nil {
			return errors.New("unknown http filter: " + name)
		}

		pos := p.Order().Position
		if pos <= plugins.OrderPositionAuthn || pos >= plugins.OrderPositionInner {
			return errors.New("http filter should not in authn/pre/post position: " + name)
		}

		data := filter.Config.Raw
		conf := p.Config()
		if err := protojson.Unmarshal(data, conf); err != nil {
			return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
		}

		if err := conf.Validate(); err != nil {
			return fmt.Errorf("invalid config for filter %s: %w", name, err)
		}
	}

	return nil
}

func ValidateServiceRegistry(sr *ServiceRegistry) error {
	reg, err := registry.CreateRegistry(sr.Spec.Type, nil, sr.ObjectMeta)
	if err != nil {
		return err
	}

	_, err = registry.ParseConfig(reg, sr.Spec.Config.Raw)
	return err
}
