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
	"github.com/spf13/viper"

	"mosn.io/htnn/pkg/log"
)

var (
	logger = log.DefaultLogger.WithName("config")
)

func GoSoPath() string {
	return "/etc/libgolang.so"
}

var rootNamespace = "istio-system"

func RootNamespace() string {
	return rootNamespace
}

func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logger.Error(err, "read config file failed")
		}

		return
	}

	logger.Info("use config file", "filename", viper.ConfigFileUsed())

	cfgRootNamespace := viper.GetString("istio.rootNamespace")
	if cfgRootNamespace != "" {
		rootNamespace = cfgRootNamespace
	}
}
