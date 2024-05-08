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

//go:build benchmark

package integration

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/integration/control_plane"
	"mosn.io/htnn/api/plugins/tests/integration/data_plane"
)

// The benchmarks here are used to profile and improve the data plane performance.
// Compare the results with other proxies or the same proxy but test in different configuration is very unscientific.

func benchmarkClient(t *testing.T) string {
	// use https://github.com/codesenberg/bombardier as benchmark client
	path, err := exec.Command("which", "bombardier").CombinedOutput()
	require.NoError(t, err, string(path))
	return strings.TrimSpace(string(path))
}

func runBenchmark(t *testing.T, clientBin string, url string) {
	duration := os.Getenv("HTNN_DATA_PLANE_BENCHMARK_DURATION")
	if duration == "" {
		duration = "15s"
	}

	args := fmt.Sprintf("--http2 -c 8 -d %s --latencies %s", duration, url)
	cli := exec.Command(clientBin, strings.Fields(args)...)
	out, err := cli.CombinedOutput()
	require.NoError(t, err, string(out))
	t.Logf("benchmark result: %s", out)
}

func runBenchmarkWithHeaders(t *testing.T, clientBin string, url string, hdr http.Header) {
	duration := os.Getenv("HTNN_DATA_PLANE_BENCHMARK_DURATION")
	if duration == "" {
		duration = "15s"
	}

	args := fmt.Sprintf("--http2 -c 8 -d %s --latencies %s", duration, url)
	hdrArgs := []string{}
	for k, v := range hdr {
		for _, vv := range v {
			hdrArgs = append(hdrArgs, fmt.Sprintf(`--header=%s:%s`, k, vv))
		}
	}

	cli := exec.Command(clientBin, append(hdrArgs, strings.Fields(args)...)...)
	out, err := cli.CombinedOutput()
	require.NoError(t, err, string(out))
	t.Logf("benchmark result: %s", out)
}

func TestBenchmarkPlugin(t *testing.T) {
	clientBin := benchmarkClient(t)

	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		LogLevel: "warn",
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	config := control_plane.NewSinglePluinConfig("benchmark", map[string]interface{}{})
	controlPlane.UseGoPluginConfig(t, config, dp)

	runBenchmark(t, clientBin, "http://localhost:10000/echo")
}

func TestBenchmarkPluginFromConsumer(t *testing.T) {
	clientBin := benchmarkClient(t)

	dp, err := data_plane.StartDataPlane(t, &data_plane.Option{
		LogLevel: "warn",
		Bootstrap: data_plane.Bootstrap().AddConsumer("marvin", map[string]interface{}{
			"auth": map[string]interface{}{
				"consumer": `{"name":"marvin"}`,
			},
			"filters": map[string]interface{}{
				"benchmark2": map[string]interface{}{
					"config": `{}`,
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	config := control_plane.NewPluinConfig([]*model.FilterConfig{
		{
			Name:   "consumer",
			Config: map[string]interface{}{},
		},
		{
			Name:   "benchmark",
			Config: map[string]interface{}{},
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	hdr := http.Header{}
	hdr.Set("Authorization", "marvin")
	runBenchmarkWithHeaders(t, clientBin, "http://localhost:10000/echo", hdr)
}
