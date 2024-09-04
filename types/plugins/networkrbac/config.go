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

package networkrbac

import (
	"fmt"

	matcher "github.com/cncf/xds/go/xds/type/matcher/v3"
	rbacconfig "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	rbac "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	"google.golang.org/protobuf/proto"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "networkRBAC"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionNetwork,
	}
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeAuthz
}

type CustomConfig struct {
	rbac.RBAC
}

var matchingNetworkInputs = []string{
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.DestinationIPInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.DestinationPortInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourcePortInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.DirectSourceIPInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceTypeInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.ServerNameInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.TransportProtocolInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.ApplicationProtocolInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.FilterStateInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.ssl.v3.UriSanInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.ssl.v3.DnsSanInput",
	"type.googleapis.com/envoy.extensions.matching.common_inputs.ssl.v3.SubjectInput",
}

func (conf *CustomConfig) Validate() error {
	err := conf.RBAC.Validate()
	if err != nil {
		return err
	}

	m := conf.RBAC.GetMatcher().GetMatcherTree()
	typeURL := m.GetInput().GetTypedConfig().GetTypeUrl()
	found := false
	for _, allowList := range matchingNetworkInputs {
		if typeURL == allowList {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("matcher.matcherTree.input.typedConfig.typeUrl must be one of %v, got %s", matchingNetworkInputs, typeURL)
	}

	if m.GetCustomMatch().GetTypedConfig() != nil {
		typeURL = m.GetCustomMatch().GetTypedConfig().GetTypeUrl()
		if typeURL != "type.googleapis.com/xds.type.matcher.v3.IPMatcher" {
			return fmt.Errorf("matcher.matcherTree.customMatch.typedConfig.typeUrl must be type.googleapis.com/xds.type.matcher.v3.IPMatcher, got %s", typeURL)
		}
		v := m.GetCustomMatch().GetTypedConfig().GetValue()
		matcher := &matcher.IPMatcher{}
		// We always call Validate after Unmarshal success
		_ = proto.Unmarshal(v, matcher)
		err := matcher.Validate()
		if err != nil {
			return err
		}

		for _, rg := range matcher.GetRangeMatchers() {
			v = rg.GetOnMatch().GetAction().GetTypedConfig().GetValue()
			action := &rbacconfig.Action{}
			_ = proto.Unmarshal(v, action)
			err := action.Validate()
			if err != nil {
				return err
			}
		}
	}
	// TODO: validate the Action in the other matcher like ExactMatchMap
	// Another TODO: do it more smartly
	return nil
}

func (p *Plugin) Config() api.PluginConfig {
	return &CustomConfig{}
}
