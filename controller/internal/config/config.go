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

package config

import (
	"strings"
	"sync"

	"github.com/spf13/viper"

	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/controller/internal/log"
)

func updateStringIfSet(vp *viper.Viper, key string, item *string) {
	if vp.IsSet(key) {
		*item = vp.GetString(key)
		return
	}
}

func updateBoolIfSet(vp *viper.Viper, key string, item *bool) {
	if vp.IsSet(key) {
		*item = vp.GetBool(key)
		return
	}
}

var (
	configLock sync.RWMutex
)

var goSoPath = "/etc/libgolang.so"

// Should match the Go shared library put in the data plane image
func GoSoPath() string {
	configLock.RLock()
	defer configLock.RUnlock()
	return goSoPath
}

var rootNamespace = "istio-system"

// Should match istio's rootNamespace configuration.
// See https://istio.io/latest/docs/reference/config/istio.mesh.v1alpha1/#MeshConfig for more info.
// This field is automatically configured when HTNN controller is run in the istiod.
func RootNamespace() string {
	configLock.RLock()
	defer configLock.RUnlock()
	return rootNamespace
}

var enableGatewayAPI = true

// If this is set to true, support for Kubernetes gateway-api will be enabled.
// In addition to this being enabled, the gateway-api CRDs need to be installed.
// This field is automatically configured when HTNN controller is run in the istiod.
func EnableGatewayAPI() bool {
	configLock.RLock()
	defer configLock.RUnlock()
	return enableGatewayAPI
}

var enableEmbeddedMode = true

// Enable embedded mode so that HTNN won't check the annotation of the target resource.
func EnableEmbeddedMode() bool {
	configLock.RLock()
	defer configLock.RUnlock()
	return enableEmbeddedMode
}

var enableNativePlugin = true

// Enable Native plugin. Sometimes we may need to disable all native plugins, because:
// 1. Only want to use Go plugins
// 2. A custom Envoy is used and it doesn't support all Envoy's http filters as the default
// open source one.
func EnableNativePlugin() bool {
	configLock.RLock()
	defer configLock.RUnlock()
	return enableNativePlugin
}

// LDS Plugin Via ECDS is disabled by default, because
// 1. Per-LDS ECDS may be expensive in some cases.
// 2. We can't disable a LDS plugin via ECDS. So every route under this LDS will execute it.
//
// You can enable it if
// 1. You are using HTNN as south-north gateway.
// 2. The number of LDS is limited. Better to run a benchmark by yourself to see if it's suitable for you.
// 3. You need LDS level plugin.
var enableLDSPluginViaECDS = false

// Enable dispatching LDS plugin via ECDS. If we dispatch LDS plugin via LDS directly, it will cause
// connection close.
func EnableLDSPluginViaECDS() bool {
	configLock.RLock()
	defer configLock.RUnlock()
	return enableLDSPluginViaECDS
}

type envStringReplacer struct {
}

func (r *envStringReplacer) Replace(s string) string {
	return strings.ReplaceAll(s, ".", "_")
}

func Init() {
	configLock.Lock()
	defer configLock.Unlock()

	vp := viper.NewWithOptions(viper.EnvKeyReplacer(&envStringReplacer{}))
	vp.SetEnvPrefix("HTNN")
	vp.AutomaticEnv()
	// a config item `envoy.go_so_path` can be set with env `HTNN_ENVOY_GO_SO_PATH`

	updateStringIfSet(vp, "envoy.go_so_path", &goSoPath)
	updateStringIfSet(vp, "istio.root_namespace", &rootNamespace)

	updateBoolIfSet(vp, "enable_gateway_api", &enableGatewayAPI)
	updateBoolIfSet(vp, "enable_embedded_mode", &enableEmbeddedMode)
	updateBoolIfSet(vp, "enable_native_plugin", &enableNativePlugin)
	updateBoolIfSet(vp, "enable_lds_plugin_via_ecds", &enableLDSPluginViaECDS)

	postInit()
}

func postInit() {
	if !enableNativePlugin {
		log.Infof("native plugin disabled by configured")
		plugins.IterateHttpPlugin(func(key string, value plugins.Plugin) bool {
			_, ok := value.(plugins.NativePlugin)
			if !ok {
				return true
			}

			plugins.DisableHttpPlugin(key)
			return true
		})

	}
}
