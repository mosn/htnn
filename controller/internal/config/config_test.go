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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setEnvForTest() {
	os.Setenv("HTNN_ENABLE_GATEWAY_API", "false")
	os.Setenv("HTNN_ENABLE_EMBEDDED_MODE", "false")
	os.Setenv("HTNN_ENABLE_NATIVE_PLUGIN", "false")
	os.Setenv("HTNN_ENVOY_GO_SO_PATH", "/usr/local/golang.so")
	os.Setenv("HTNN_ISTIO_ROOT_NAMESPACE", "htnn")
	os.Setenv("HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS", "true")
	os.Setenv("HTNN_USE_WILDCARD_IPV6_IN_LDS_NAME", "true")
}

func TestInit(t *testing.T) {
	Init()

	// Check default values
	assert.Equal(t, true, EnableGatewayAPI())
	assert.Equal(t, true, EnableEmbeddedMode())
	assert.Equal(t, true, EnableNativePlugin())
	assert.Equal(t, "/etc/libgolang.so", GoSoPath())
	assert.Equal(t, "istio-system", RootNamespace())
	assert.Equal(t, false, EnableLDSPluginViaECDS())
	assert.Equal(t, false, UseWildcardIPv6InLDSName())

	setEnvForTest()
	Init()

	assert.Equal(t, false, EnableGatewayAPI())
	assert.Equal(t, false, EnableEmbeddedMode())
	assert.Equal(t, false, EnableNativePlugin())
	assert.Equal(t, "/usr/local/golang.so", GoSoPath())
	assert.Equal(t, "htnn", RootNamespace())
	assert.Equal(t, true, EnableLDSPluginViaECDS())
	assert.Equal(t, true, UseWildcardIPv6InLDSName())
}
