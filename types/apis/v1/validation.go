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

	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/pkg/proto"
	"mosn.io/htnn/types/pkg/registry"
	_ "mosn.io/htnn/types/plugins"    // register plugin types
	_ "mosn.io/htnn/types/registries" // register registry types
)

// ValidateHTTPFilterPolicy validates HTTPFilterPolicy.
// It only validate the part it knows, so unknown plugins or fields will be skipped.
// It's recommended to use this function in the controller.
func ValidateHTTPFilterPolicy(policy *HTTPFilterPolicy) error {
	return validateHTTPFilterPolicy(policy, false)
}

// ValidateHTTPFilterPolicyStrictly validates HTTPFilterPolicy strictly.
// Unknown plugins or fields will be rejected.
// It's recommended to use this function before writing the configuration to persistent storage,
// for example, in the dashboard or webhook.
func ValidateHTTPFilterPolicyStrictly(policy *HTTPFilterPolicy) error {
	return validateHTTPFilterPolicy(policy, true)
}

func validateFilter(name string, filter HTTPPlugin, strict bool, targetGateway bool) error {
	p := plugins.LoadHttpPluginType(name)
	if p == nil {
		if strict {
			return errors.New("unknown http filter: " + name)
		}
		return nil
	}

	if targetGateway {
		switch p.Order().Position {
		case plugins.OrderPositionOuter, plugins.OrderPositionInner:
			// We can't directly provide different ECDS for every native plugins. There will
			// be more than 20 native plugins in the future, and it's not reasonable to provide
			// such number (20 x the number of LDS) of ECDS resources. Perhaps we can use
			// composite filter to solve this problem?
			return errors.New("configure native plugins to the Gateway is not implemented")
		}
	}

	data := filter.Config.Raw
	conf := p.Config()
	var err error
	if strict {
		err = proto.UnmarshalJSONStrictly(data, conf)
	} else {
		err = proto.UnmarshalJSON(data, conf)
	}
	if err != nil {
		return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
	}

	if err := conf.Validate(); err != nil {
		return fmt.Errorf("invalid config for filter %s: %w", name, err)
	}
	return nil
}

func validateHTTPFilterPolicy(policy *HTTPFilterPolicy, strict bool) error {
	targetGateway := false
	ref := policy.Spec.TargetRef
	if ref != nil {
		if ref.Namespace != nil {
			namespace := string(*ref.Namespace)
			if namespace != policy.Namespace {
				return errors.New("namespace in TargetRef doesn't match HTTPFilterPolicy's namespace")
			}
		}

		if ref.SectionName != nil {
			if len(policy.Spec.SubPolicies) > 0 {
				return errors.New("targetRef.SectionName and SubPolicies can not be used together")
			}
		}

		validTarget := false
		if ref.Group == "networking.istio.io" {
			switch ref.Kind {
			case "VirtualService":
				validTarget = true
			case "Gateway":
				// To target HTTPFilterPolicy to Gateway, ensure environment variable "HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS"
				// is set to "true" in the controller.
				// Note that the Gateway support may be different than you think. As we attach the generated ECDS to the
				// LDS, and istio will merge multiple Gateways which listen on the same port to a LDS, so the HTTPFilterPolicy
				// will not just target one Gateway if you have multiple Gateway which listen on the same port but have different hostname.
				//
				// TODO: implement the Gateway support via RDS, so it matches the model 100%.
				validTarget = true
			}
		} else if ref.Group == "gateway.networking.k8s.io" {
			switch ref.Kind {
			case "HTTPRoute", "Gateway":
				validTarget = true
			}
		}
		if !validTarget {
			return errors.New("unsupported targetRef.group or targetRef.kind")
		}

		targetGateway = ref.Kind == "Gateway"
	}
	// HTTPFilterPolicy in embedded mode can have no targetRef

	for name, filter := range policy.Spec.Filters {
		err := validateFilter(name, filter, strict, targetGateway)
		if err != nil {
			return err
		}
	}

	for _, policy := range policy.Spec.SubPolicies {
		for name, filter := range policy.Filters {
			err := validateFilter(name, filter, strict, targetGateway)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func ValidateVirtualService(vs *istiov1a3.VirtualService) error {
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

func NormalizeIstioProtocol(protocol string) string {
	return strings.ToUpper(protocol)
}

func ValidateGateway(gw *istiov1a3.Gateway) error {
	for i, svr := range gw.Spec.Servers {
		if svr.Port == nil {
			return fmt.Errorf("spec.servers[%d].port: Required value", i)
		}
		proto := NormalizeIstioProtocol(svr.Port.Protocol)
		if proto != "HTTP" && proto != "HTTPS" {
			return fmt.Errorf("spec.servers[%d].port.protocol: Only HTTP and HTTPS are supported", i)
		}

		for _, host := range svr.Hosts {
			if strings.ContainsRune(host, '/') {
				// TODO: support it
				return errors.New("Gateway has host with namespace is not supported")
			}
		}
	}
	return nil
}

func NormalizeK8sGatewayProtocol(protocol gwapiv1.ProtocolType) string {
	// So far, all k8s gateway protocols are valid istio protocols
	return NormalizeIstioProtocol(string(protocol))
}

func ValidateConsumer(c *Consumer) error {
	for name, filter := range c.Spec.Auth {
		plugin := plugins.LoadHttpPluginType(name)
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
		if err := proto.UnmarshalJSON(data, conf); err != nil {
			return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
		}

		if err := conf.Validate(); err != nil {
			return fmt.Errorf("invalid config for filter %s: %w", name, err)
		}
	}

	for name, filter := range c.Spec.Filters {
		p := plugins.LoadHttpPluginType(name)
		if p == nil {
			return errors.New("unknown http filter: " + name)
		}

		pos := p.Order().Position
		if pos <= plugins.OrderPositionAuthn || pos >= plugins.OrderPositionInner {
			return errors.New("http filter should not in authn/pre/post position: " + name)
		}

		data := filter.Config.Raw
		conf := p.Config()
		if err := proto.UnmarshalJSON(data, conf); err != nil {
			return fmt.Errorf("failed to unmarshal for filter %s: %w", name, err)
		}

		if err := conf.Validate(); err != nil {
			return fmt.Errorf("invalid config for filter %s: %w", name, err)
		}
	}

	return nil
}

func ValidateServiceRegistry(sr *ServiceRegistry) error {
	reg := registry.GetRegistryType(sr.Spec.Type)
	if reg == nil {
		return fmt.Errorf("unknown registry type: %s", sr.Spec.Type)
	}

	_, err := registry.ParseConfig(reg, sr.Spec.Config.Raw)
	return err
}
