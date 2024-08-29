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

package listenerpatch

import (
	"fmt"

	envoyapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	file "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	grpc "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	opentelemetry "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/open_telemetry/v3"
	stream "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	wasm "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/wasm/v3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "listenerPatch"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionListener,
	}
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeGeneral
}

func (p *Plugin) Config() api.PluginConfig {
	return &CustomConfig{}
}

type CustomConfig struct {
	envoyapi.Listener
}

type config interface {
	protoreflect.ProtoMessage

	Validate() error
	Reset()
}

var (
	loggerTypedConfigs = map[string]config{
		"type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog":                          &file.FileAccessLog{},
		"type.googleapis.com/envoy.extensions.access_loggers.open_telemetry.v3.OpenTelemetryAccessLogConfig": &opentelemetry.OpenTelemetryAccessLogConfig{},
		"type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog":                      &stream.StdoutAccessLog{},
		"type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StderrAccessLog":                      &stream.StderrAccessLog{},
		"type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.HttpGrpcAccessLogConfig":                &grpc.HttpGrpcAccessLogConfig{},
		"type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.TcpGrpcAccessLogConfig":                 &grpc.TcpGrpcAccessLogConfig{},
		"type.googleapis.com/envoy.extensions.access_loggers.wasm.v3.WasmAccessLog":                          &wasm.WasmAccessLog{},
	}
)

func (conf *CustomConfig) Validate() error {
	// We can't use the default validation because the listener is not a full listener.
	for _, logger := range conf.Listener.AccessLog {
		tc := logger.GetTypedConfig()
		if tc == nil {
			return fmt.Errorf("access log config is nil")
		}

		typeURL := tc.TypeUrl
		cfg, ok := loggerTypedConfigs[typeURL]
		if !ok {
			return fmt.Errorf("unknown logger type: %s", typeURL)
		}

		// We always call Validate after Unmarshal success
		_ = proto.Unmarshal(tc.GetValue(), cfg)

		err := cfg.Validate()
		cfg.Reset()
		if err != nil {
			return err
		}
	}

	// TODO: support other fields

	return nil
}
