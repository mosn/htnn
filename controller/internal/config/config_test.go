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
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	Init()

	// Check default values
	assert.Equal(t, "istio-system", RootNamespace())
	assert.Equal(t, "127.0.0.1:15110", McpServerListenAddress())

	viper.AddConfigPath("./testdata")
	Init()

	assert.Equal(t, "htnn", RootNamespace())
	assert.Equal(t, ":9989", McpServerListenAddress())
}
