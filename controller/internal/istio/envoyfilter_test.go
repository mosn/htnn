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

package istio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	local_ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/local_ratelimit/v3"
	"github.com/stretchr/testify/require"
	istiov1a3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/yaml"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	ctrlcfg "mosn.io/htnn/controller/internal/config"
)

type basePlugin struct {
}

func (p *basePlugin) RouteConfigTypeURL() string {
	return "type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit"
}

func (p basePlugin) Config() api.PluginConfig {
	return &local_ratelimit.LocalRateLimit{}
}

type pluginFirst struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *pluginFirst) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position:  plugins.OrderPositionOuter,
		Operation: plugins.OrderOperationInsertFirst,
	}
}

func (p *pluginFirst) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":      p.RouteConfigTypeURL(),
			"statPrefix": "http_local_rate_limiter",
		},
	}
}

type pluginPre struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *pluginPre) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionOuter,
	}
}

func (p *pluginPre) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":      p.RouteConfigTypeURL(),
			"statPrefix": "http_local_rate_limiter",
		},
	}
}

type pluginPost struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *pluginPost) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionInner,
	}
}

func (p *pluginPost) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":      p.RouteConfigTypeURL(),
			"statPrefix": "http_local_rate_limiter",
		},
	}
}

type pluginLast struct {
	plugins.PluginMethodDefaultImpl
	basePlugin
}

func (p *pluginLast) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position:  plugins.OrderPositionInner,
		Operation: plugins.OrderOperationInsertLast,
	}
}

func (p *pluginLast) HTTPFilterConfigPlaceholder() map[string]interface{} {
	return map[string]interface{}{
		"typed_config": map[string]interface{}{
			"@type":      p.RouteConfigTypeURL(),
			"statPrefix": "http_local_rate_limiter",
		},
	}
}

func TestDefaultFilters(t *testing.T) {
	patch := gomonkey.ApplyFuncReturn(ctrlcfg.GoSoPath, "/path/to/goso")
	defer patch.Reset()

	plugins.RegisterHttpPlugin("first", &pluginFirst{})
	plugins.RegisterHttpPlugin("pre", &pluginPre{})
	plugins.RegisterHttpPlugin("post", &pluginPost{})
	plugins.RegisterHttpPlugin("last", &pluginLast{})

	out := []*istiov1a3.EnvoyFilter{}
	for _, ef := range DefaultEnvoyFilters() {
		out = append(out, ef)
	}
	d, _ := yaml.Marshal(out)
	actual := string(d)
	expFile := filepath.Join("testdata", "default_filters.yml")
	d, _ = os.ReadFile(expFile)
	want := string(d)
	require.Equal(t, want, actual)
}

func TestGenerateConsumers(t *testing.T) {
	patch := gomonkey.ApplyFuncReturn(ctrlcfg.GoSoPath, "/path/to/goso")
	defer patch.Reset()

	out := GenerateConsumers(map[string]interface{}{
		"ns": map[string]interface{}{
			"consumer1": "config",
			"consumer2": "config",
		},
	})
	d, _ := yaml.Marshal(out)
	actual := string(d)
	expFile := filepath.Join("testdata", "consumers.yml")
	d, _ = os.ReadFile(expFile)
	want := string(d)
	require.Equal(t, want, actual)
}
