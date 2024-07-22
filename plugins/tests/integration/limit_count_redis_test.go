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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

func TestLimitCountRedis(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":6379", "redis")

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "sanity",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "e27e2f7f",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("X-Envoy-Ratelimited"))
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
				assert.Equal(t, "true", resp.Header.Get("X-Envoy-Ratelimited"))
				resp, _ = dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)

				time.Sleep(1 * time.Second)
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
		{
			name: "multiple rules",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "cd81da10",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
					map[string]interface{}{
						"count":      2,
						"timeWindow": "1s",
					},
					map[string]interface{}{
						"count":      3,
						"timeWindow": "1s",
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", nil)
				// each rule counts separately
				assert.Equal(t, 200, resp.StatusCode)
				hdr2 := http.Header{}
				hdr2.Add("x-key", "2")
				resp, _ = dp.Head("/echo", hdr2)
				assert.Equal(t, 429, resp.StatusCode)

				time.Sleep(1 * time.Second)
				resp, _ = dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
		{
			name: "single rule, with limit quota headers enabled",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":                  "24e24b12",
				"address":                 "redis:6379",
				"enableLimitQuotaHeaders": true,
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, []string{"1, 1;w=1"}, resp.Header.Values("X-Ratelimit-Limit"))
				assert.Equal(t, []string{"0"}, resp.Header.Values("X-Ratelimit-Remaining"))
				assert.Equal(t, []string{"1"}, resp.Header.Values("X-Ratelimit-Reset"))
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
				assert.Equal(t, "1, 1;w=1", resp.Header.Get("X-Ratelimit-Limit"))
				assert.Equal(t, "0", resp.Header.Get("X-Ratelimit-Remaining"))
				assert.Equal(t, "1", resp.Header.Get("X-Ratelimit-Reset"))
			},
		},
		{
			name: "multiple rules, with limit quota headers enabled",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":                  "2ce0ecd7",
				"address":                 "redis:6379",
				"enableLimitQuotaHeaders": true,
				"rules": []interface{}{
					map[string]interface{}{
						"count":      2,
						"timeWindow": "10s",
						"key":        `request.header("x-key")`,
					},
					map[string]interface{}{
						"count":      2,
						"timeWindow": "1s",
					},
					map[string]interface{}{
						"count":      3,
						"timeWindow": "1s",
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, []string{"2, 2;w=10, 2;w=1, 3;w=1"}, resp.Header.Values("X-Ratelimit-Limit"))
				assert.Equal(t, []string{"1"}, resp.Header.Values("X-Ratelimit-Remaining"))
				assert.Equal(t, []string{"10"}, resp.Header.Values("X-Ratelimit-Reset"))

				hdr2 := http.Header{}
				hdr2.Add("x-key", "2")
				resp, _ = dp.Head("/echo", hdr2)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "2, 2;w=10, 2;w=1, 3;w=1", resp.Header.Get("X-Ratelimit-Limit"))
				assert.Equal(t, "0", resp.Header.Get("X-Ratelimit-Remaining"))
				assert.Equal(t, "1", resp.Header.Get("X-Ratelimit-Reset"))

				resp, _ = dp.Head("/echo", nil)
				assert.Equal(t, 429, resp.StatusCode)
				assert.Equal(t, "2, 2;w=10, 2;w=1, 3;w=1", resp.Header.Get("X-Ratelimit-Limit"))
				assert.Equal(t, "0", resp.Header.Get("X-Ratelimit-Remaining"))
				assert.Equal(t, "1", resp.Header.Get("X-Ratelimit-Reset"))
			},
		},
		{
			name: "passwd",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "ff100fd4",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"username": "user",
				"password": "passwd",
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
			},
		},
		{
			name: "tls",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "f7ce8fed",
				"address": "redis:6380",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"tls":           true,
				"tlsSkipVerify": true,
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
			},
		},
		{
			name: "rateLimitedStatus",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "b52c8aee",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"rateLimitedStatus": 503,
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 503, resp.StatusCode)
			},
		},
		{
			name: "rateLimitedStatus < 400",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "56102bea",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"rateLimitedStatus": 200,
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
			},
		},
		{
			name: "keep counter across rds update (part 1)",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "f7ce8fcc",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "10s",
						"key":        `request.header("x-key")`,
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("X-Envoy-Ratelimited"))
			},
		},
		{
			name: "keep counter across rds update (part 2)",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "f7ce8fcc",
				"address": "redis:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "10s",
						"key":        `request.header("x-key")`,
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
				assert.Equal(t, "true", resp.Header.Get("X-Envoy-Ratelimited"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			tt.run(t)
		})
	}
}

func TestLimitCountRedisBadService(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		NoErrorLogCheck: true,
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "bad redis",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "6e1643e9",
				"address": "redisx:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
		{
			name: "bad redis, failure mode deny",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":          "22e1afc9",
				"address":         "redisx:6379",
				"failureModeDeny": true,
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 500, resp.StatusCode)
			},
		},
		{
			name: "statusOnError",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "02945c93",
				"address": "redisx:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"failureModeDeny": true,
				"statusOnError":   502,
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 502, resp.StatusCode)
			},
		},
		{
			name: "statusOnError, no failureModeDeny",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "efa91d02",
				"address": "redisx:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"statusOnError": 502,
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
		{
			name: "bad redis, wrong passwd",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":          "f446d349",
				"address":         "redis:6379",
				"username":        "user",
				"password":        "x",
				"failureModeDeny": true,
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
					},
				},
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 500, resp.StatusCode)
			},
		},
		{
			name: "bad redis, tls verify",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":          "6e4fca85",
				"address":         "redis:6380",
				"failureModeDeny": true,
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
					},
				},
				"tls": true,
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 500, resp.StatusCode)
			},
		},
		{
			name: "bad redis, don't produce limit quota headers",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix":  "7f1cf524",
				"address": "redisx:6379",
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
					},
				},
				"enableLimitQuotaHeaders": true,
			}),
			run: func(t *testing.T) {
				resp, _ := dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			tt.run(t)
		})
	}
}

func TestLimitCountRedisClusterMode(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":6400", "redis-cluster")

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "single rules",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix": "c8e8eeb9",
				"cluster": map[string]interface{}{
					"addresses": []interface{}{
						"redis-cluster-0:6379",
						"redis-cluster-1:6379",
						"redis-cluster-2:6379",
					},
				},
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"tls":           true,
				"tlsSkipVerify": true,
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", hdr)
				assert.Equal(t, 429, resp.StatusCode)
			},
		},
		{
			name: "multiple rules",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix": "1d5db164",
				"cluster": map[string]interface{}{
					"addresses": []interface{}{
						"redis-cluster-0:6379",
						"redis-cluster-1:6379",
						"redis-cluster-2:6379",
					},
				},
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
					map[string]interface{}{
						"count":      2,
						"timeWindow": "1s",
					},
					map[string]interface{}{
						"count":      3,
						"timeWindow": "1s",
					},
				},
				"tls":           true,
				"tlsSkipVerify": true,
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 200, resp.StatusCode)
				resp, _ = dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)

				hdr2 := http.Header{}
				hdr2.Add("x-key", "2")
				resp, _ = dp.Head("/echo", hdr2)
				assert.Equal(t, 429, resp.StatusCode)

				time.Sleep(1 * time.Second)
				resp, _ = dp.Head("/echo", nil)
				assert.Equal(t, 200, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			tt.run(t)
		})
	}
}

func TestLimitCountRedisClusterModeBadService(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		NoErrorLogCheck: true,
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":6400", "redis-cluster")

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "failure mode deny",
			config: controlplane.NewSinglePluinConfig("limitCountRedis", map[string]interface{}{
				"prefix": "e2cbf683",
				"cluster": map[string]interface{}{
					"addresses": []interface{}{
						"redis-cluster-0:6379",
						"redis-cluster-1:6379",
						"redis-cluster-2:6379",
					},
				},
				"rules": []interface{}{
					map[string]interface{}{
						"count":      1,
						"timeWindow": "1s",
						"key":        `request.header("x-key")`,
					},
				},
				"failureModeDeny": true,
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("x-key", "1")
				resp, _ := dp.Head("/echo", hdr)
				assert.Equal(t, 500, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			tt.run(t)
		})
	}
}
