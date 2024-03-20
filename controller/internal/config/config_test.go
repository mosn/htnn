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

func TestInit(t *testing.T) {
	Init()

	// Check default values
	assert.Equal(t, false, EnableWebhooks())
	assert.Equal(t, false, EnableGatewayAPI())
	assert.Equal(t, "/etc/libgolang.so", GoSoPath())
	assert.Equal(t, "istio-system", RootNamespace())

	os.Chdir("./testdata")
	Init()
	os.Chdir("..")

	assert.Equal(t, true, EnableWebhooks())
	assert.Equal(t, true, EnableGatewayAPI())
	assert.Equal(t, "/usr/local/golang.so", GoSoPath())
	assert.Equal(t, "htnn", RootNamespace())
}
