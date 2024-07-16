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

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/plugins/tests/integration/control_plane"
	"mosn.io/htnn/api/plugins/tests/integration/data_plane"
)

func TestLogLevelCache(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		LogLevel: "error",
		Envs: map[string]string{
			"ENVOY_GOLANG_LOG_LEVEL_SYNC_INTERVAL": "10ms",
		},
		NoErrorLogCheck: true,
		ExpectLogPattern: []string{
			// Logs below is in debug level log. In this test we will change the log level and
			// check if it takes effect.
			`run plugin buffer, method: DecodeHeaders`,
		},
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	config := control_plane.NewSinglePluinConfig("buffer", map[string]interface{}{})
	controlPlane.UseGoPluginConfig(t, config, dp)
	err = dp.SetLogLevel("golang", "debug")
	require.Nil(t, err)
	time.Sleep(100 * time.Millisecond) // wait for log level syncer take effect

	resp, err := dp.Get("/echo", nil)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, resp)
}
