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

	"github.com/spf13/viper"

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

var goSoPath = "/etc/libgolang.so"

func GoSoPath() string {
	return goSoPath
}

var rootNamespace = "istio-system"

func RootNamespace() string {
	return rootNamespace
}

var enableGatewayAPI = true

func EnableGatewayAPI() bool {
	return enableGatewayAPI
}

var enableEmbeddedMode = true

func EnableEmbeddedMode() bool {
	return enableEmbeddedMode
}

type envStringReplacer struct {
}

func (r *envStringReplacer) Replace(s string) string {
	return strings.ReplaceAll(s, ".", "_")
}

func Init() {
	vp := viper.NewWithOptions(viper.EnvKeyReplacer(&envStringReplacer{}))
	vp.SetEnvPrefix("HTNN")
	vp.AutomaticEnv()
	// a config item `envoy.go_so_path` can be set with env `HTNN_ENVOY_GO_SO_PATH`, which is prior to the value in config file

	vp.SetConfigName("config")
	vp.SetConfigType("yaml")
	vp.AddConfigPath(".")
	vp.AddConfigPath("./config")

	if err := vp.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Errorf("read config file failed, err: %v", err)
		}
	} else {
		log.Infof("use config file [%s]", vp.ConfigFileUsed())
	}

	updateStringIfSet(vp, "envoy.go_so_path", &goSoPath)
	updateStringIfSet(vp, "istio.root_namespace", &rootNamespace)

	updateBoolIfSet(vp, "enable_gateway_api", &enableGatewayAPI)
	updateBoolIfSet(vp, "enable_embedded_mode", &enableEmbeddedMode)
}
