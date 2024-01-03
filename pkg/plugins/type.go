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

package plugins

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"mosn.io/htnn/pkg/filtermanager/api"
)

type PluginType int

const (
	TypeSecurity      PluginType = iota // Plugins like WAF, request validation, etc.
	TypeAuthn                           // Plugins do authentication
	TypeAuthz                           // Plugins do authorization
	TypeTraffic                         // Plugins do traffic control, like rate limit, circuit breaker, etc.
	TypeTransform                       // Plugins do request/response transform
	TypeObservability                   // Plugins do observability
	TypeGeneral
)

// PluginOrder is used by the control plane to specify the order of the plugins, especially during merging.
// There is always a requirement to specify the order by users.
// For now, we just provide a default order in plugins. Therefore, users don't need to manually configure the order.
// In future, we can let the users to specify a global order or a relative order in some plugins.
// Note that the order is strictly followed only when the plugins are run in DecodeHeaders and Log.
// To know the details, please refer to:
// https://github.com/mosn/htnn/blob/main/content/en/docs/developer-guide/plugin_development.md

type PluginOrderPosition int

const (
	OrderPositionPre PluginOrderPosition = iota // First position. It's reserved for Native plugins.

	// Now goes the Go plugins
	OrderPositionAccess
	OrderPositionAuthn
	OrderPositionAuthz
	OrderPositionTraffic
	OrderPositionTransform
	OrderPositionUnspecified
	OrderPositionBeforeUpstream

	// Stats plugins are expected to be called mainly in the Log phase
	OrderPositionStats

	// Istio's extensions go here

	// Last position. It's reserved for Native plugins.
	OrderPositionPost
)

type PluginOrderOperation int

// If InsertFirst is specified, the plugin will be ordered from the beginning of the group.
// InsertLast is the opposite.
const (
	OrderOperationInsertFirst PluginOrderOperation = -1
	OrderOperationNop         PluginOrderOperation = 0 // Nop is the default
	OrderOperationInsertLast  PluginOrderOperation = 1
)

type PluginOrder struct {
	Position  PluginOrderPosition
	Operation PluginOrderOperation
}

type Plugin interface {
	Config() PluginConfig

	// Optional methods
	Type() PluginType
	Order() PluginOrder
	Merge(parent interface{}, child interface{}) interface{}
}

type PluginConfig interface {
	ProtoReflect() protoreflect.Message
	Validate() error
}

type Initer interface {
	Init(cb api.ConfigCallbackHandler) error
}

type PluginConsumerConfig interface {
	PluginConfig
	Index() string
}

type NativePlugin interface {
	Plugin

	RouteConfigTypeURL() string
	DefaultHTTPFilterConfig() map[string]interface{}
}

type GoPlugin interface {
	Plugin

	ConfigFactory() api.FilterConfigFactory
}

type ConsumerPlugin interface {
	Plugin

	ConsumerConfig() PluginConsumerConfig
}
