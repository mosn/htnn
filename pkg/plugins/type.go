package plugins

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"mosn.io/moe/pkg/filtermanager/api"
)

type PluginType int

const (
	TypeSecurity      PluginType = iota // Plugins like WAF, request validation, etc.
	TypeAuthn                           // Plugins do authentication
	TypeAuthz                           // Plugins do authorization
	TypeTraffic                         // Plugins do traffic control
	TypeTransform                       // Plugins do request/response transform
	TypeObservibility                   // Plugins do observibility
	TypeGeneral
)

// PluginOrder is used by the control plane to specify the order of the plugins, especially during merging.
// There is always a requirement to specify the order by users.
// For now, we just provide a default order in plugins. Therefore, users don't need to manually configure the order.
// In future, we can let the users to specify a global order or a relative order in some plugins.
// Note that the order is strictly followed only when the plugins are run in DecodeHeaders and Log.
// To know the details, please refer to:
// https://github.com/mosn/moe/blob/main/docs/plugin_development.md

type PluginOrderPosition int

const (
	OrderPositionPre PluginOrderPosition = iota // First position. It's reserved for C++ plugins.

	// Now goes the Go plugins
	OrderPositionAccess
	OrderPositionAuthn
	OrderPositionAuthz
	OrderPositionTraffic
	OrderPositionTransform
	OrderPositionUnspecified
	OrderPositionBeforeUpstream

	// Stats plugins are expected to be called only in the Log phase
	OrderPositionStats

	// Last position. It's reserved for C++ plugins.
	OrderPositionPost

	// Istio's extensions goes here
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
	ConfigFactory() api.FilterConfigFactory

	// Optional methods
	Type() PluginType
	Order() PluginOrder
	Merge(parent interface{}, child interface{}) interface{}
}

type PluginConfig interface {
	ProtoReflect() protoreflect.Message
	Validate() error
	Init(cb api.ConfigCallbackHandler) error
}
