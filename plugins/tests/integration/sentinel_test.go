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
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
)

var (
	//go:embed sentinel_route.yaml
	sentinelRoute string
)

func doGet(respStatus int, header http.Header, query url.Values) (*http.Response, error) {
	u := fmt.Sprintf("http://localhost:10000/sentinel/status/%d", respStatus)
	if query != nil {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header = header
	tr := &http.Transport{DialContext: func(ctx context.Context, proto, addr string) (conn net.Conn, err error) {
		return net.DialTimeout("tcp", ":10000", 1*time.Second)
	}}

	client := &http.Client{Transport: tr,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	return resp, err
}

func TestSentinelFlow(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddBackendRoute(sentinelRoute),
	})
	if err != nil {
		t.Fatalf("failed to start dataplane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "simple",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"flow": map[string]interface{}{
					"rules": []interface{}{
						map[string]interface{}{
							"resource":               "f1",
							"tokenCalculateStrategy": "DIRECT",
							"controlBehavior":        "REJECT",
							"threshold":              1,
							"statIntervalInMs":       1000,
							"blockResponse": map[string]interface{}{
								"message":    "custom block resp: f1",
								"statusCode": 503,
								"headers": map[string]string{
									"X-Sentinel-Blocked": "true",
								},
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "f1")

				resp, err := doGet(200, hdr, nil)
				assert.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("X-Sentinel-Blocked"))

				resp, err = doGet(200, hdr, nil)
				assert.NoError(t, err)
				assert.Equal(t, 503, resp.StatusCode)
				assert.Equal(t, "true", resp.Header.Get("X-Sentinel-Blocked"))
				b, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				assert.Equal(t, "{\"msg\":\"custom block resp: f1\"}", string(b))

				time.Sleep(1100 * time.Millisecond)

				resp, err = doGet(200, hdr, nil)
				assert.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("X-Sentinel-Blocked"))
			},
		},
		{
			name: "resource from query",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "QUERY",
					"key":  "query",
				},
				"flow": map[string]interface{}{
					"rules": []interface{}{
						map[string]interface{}{
							"resource":               "f2",
							"tokenCalculateStrategy": "DIRECT",
							"controlBehavior":        "REJECT",
							"threshold":              1,
							"statIntervalInMs":       1000,
							"blockResponse": map[string]interface{}{
								"message":    "custom block resp: f2",
								"statusCode": 503,
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				query := url.Values{}
				query.Set("query", "f2")
				resp, _ := doGet(200, nil, query)
				assert.Equal(t, 200, resp.StatusCode)

				resp, _ = doGet(200, nil, query)
				assert.Equal(t, 503, resp.StatusCode)
				b, _ := io.ReadAll(resp.Body)
				assert.Equal(t, "{\"msg\":\"custom block resp: f2\"}", string(b))
			},
		},
		{
			name: "throttling mode",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"flow": map[string]interface{}{
					"rules": []interface{}{
						map[string]interface{}{
							"resource":               "f3",
							"tokenCalculateStrategy": "DIRECT",
							"controlBehavior":        "THROTTLING",
							"threshold":              10,
							"statIntervalInMs":       1000,
							"maxQueueingTimeMs":      1000,
							"blockResponse": map[string]interface{}{
								"statusCode": 503,
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "f3")
				m := make(map[int64]int)
				mLock := sync.Mutex{}
				for i := 0; i < 20; i++ {
					resp, _ := doGet(200, hdr, nil)
					if resp.StatusCode == 200 {
						k := time.Now().UnixMilli() / 100 // interval is 100ms
						mLock.Lock()
						m[k]++
						mLock.Unlock()
					}
				}
				for _, v := range m {
					assert.LessOrEqual(t, v, 2) // permissible req +1
				}
			},
		},
		{
			name: "associated resource",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"flow": map[string]interface{}{
					"rules": []interface{}{
						map[string]interface{}{
							"resource":               "f4",
							"tokenCalculateStrategy": "DIRECT",
							"controlBehavior":        "REJECT",
							"threshold":              1,
							"statIntervalInMs":       1000,
							"blockResponse": map[string]interface{}{
								"message":    "custom block resp: f4",
								"statusCode": 503,
							},
						},
						map[string]interface{}{
							"resource": "f5",
							"blockResponse": map[string]interface{}{
								"message":    "custom block resp: f5",
								"statusCode": 503,
							},
							"relationStrategy": "ASSOCIATED_RESOURCE",
							"refResource":      "f4",
						},
					},
				},
			}),
			run: func(t *testing.T) {
				// f4 start
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "f4")

				resp, _ := doGet(200, hdr, nil)
				assert.Equal(t, 200, resp.StatusCode)

				resp, _ = doGet(200, hdr, nil)
				assert.Equal(t, 503, resp.StatusCode)
				b, err := io.ReadAll(resp.Body)
				assert.NoError(t, err)
				assert.Equal(t, "{\"msg\":\"custom block resp: f4\"}", string(b))
				// f4 end

				// f5 start
				hdr = http.Header{}
				hdr.Add("X-Sentinel", "f5")

				// when f3 triggers the traffic limiting, f4 also triggers the traffic limiting,
				// because the two resources are related
				resp, _ = doGet(200, hdr, nil)
				assert.Equal(t, 503, resp.StatusCode)
				b, err = io.ReadAll(resp.Body)
				assert.NoError(t, err)
				assert.Equal(t, "{\"msg\":\"custom block resp: f5\"}", string(b))
				// f5 end
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

func TestSentinelHotSpot(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddBackendRoute(sentinelRoute),
	})
	if err != nil {
		t.Fatalf("failed to start dataplane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "simple",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"hotSpot": map[string]interface{}{
					"params": []string{"123", "abc", "hello"},
					"rules": []interface{}{
						map[string]interface{}{
							"resource":        "hs1",
							"metricType":      "QPS",
							"controlBehavior": "REJECT",
							"paramIndex":      0,
							"threshold":       1,
							"durationInSec":   1,
							"blockResponse": map[string]interface{}{
								"message":    "custom block resp: hs1",
								"statusCode": 503,
								"headers": map[string]string{
									"X-Sentinel-Blocked": "true",
								},
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "hs1")

				resp, err := doGet(200, hdr, nil)
				assert.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("X-Sentinel-Blocked"))

				resp, err = doGet(200, hdr, nil)
				assert.NoError(t, err)
				assert.Equal(t, 503, resp.StatusCode)
				assert.Equal(t, "true", resp.Header.Get("X-Sentinel-Blocked"))
				b, _ := io.ReadAll(resp.Body)
				assert.Equal(t, "{\"msg\":\"custom block resp: hs1\"}", string(b))

				time.Sleep(1100 * time.Millisecond)

				resp, err = doGet(200, hdr, nil)
				assert.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("X-Sentinel-Blocked"))
			},
		},
		{
			name: "with attachments",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"hotSpot": map[string]interface{}{
					"attachments": []interface{}{
						map[string]interface{}{
							"from": "HEADER",
							"key":  "X-A1",
						},
						map[string]interface{}{
							"from": "QUERY",
							"key":  "a2",
						},
					},
					"rules": []interface{}{
						map[string]interface{}{
							"resource":        "hs2",
							"metricType":      "QPS",
							"controlBehavior": "REJECT",
							"paramKey":        "X-A1", // it should actually be called `attachmentKey`
							"threshold":       1,
							"durationInSec":   1,
							"blockResponse": map[string]interface{}{
								"statusCode": 503,
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				// attachment from header X-A1
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "hs2")
				hdr.Add("X-A1", "test")

				resp, _ := doGet(200, hdr, nil)
				assert.Equal(t, 200, resp.StatusCode)

				resp, _ = doGet(200, hdr, nil)
				assert.Equal(t, 503, resp.StatusCode)

				// attachment from query a2, but attachment key is `X-A1`, so there's no traffic control
				hdr = http.Header{}
				hdr.Add("X-Sentinel", "hs2")
				query := url.Values{}
				query.Add("a2", "test")
				for i := 0; i < 5; i++ {
					resp, _ = doGet(200, hdr, query)
					assert.Equal(t, 200, resp.StatusCode)
				}
			},
		},
		{
			name: "specific items",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"hotSpot": map[string]interface{}{
					"params": []string{"123", "abc", "hello"},
					"rules": []interface{}{
						map[string]interface{}{
							"resource":        "hs3",
							"metricType":      "QPS",
							"controlBehavior": "REJECT",
							"paramIndex":      1,
							"threshold":       10,
							"durationInSec":   1,
							"specificItems":   map[string]int64{"abc": 1},
							"blockResponse": map[string]interface{}{
								"statusCode": 503,
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "hs3")

				resp, _ := doGet(200, hdr, nil)
				assert.Equal(t, 200, resp.StatusCode)

				// although the threshold is 10, the threshold for `abc` is specified to be 1 through `specificItems`
				resp, _ = doGet(200, hdr, nil)
				assert.Equal(t, 503, resp.StatusCode)
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

func TestSentinelCircuitBreaker(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddBackendRoute(sentinelRoute),
	})
	if err != nil {
		t.Fatalf("failed to start dataplane: %v", err)
		return
	}
	defer dp.Stop()

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		run    func(t *testing.T)
	}{
		{
			name: "simple",
			config: controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
				"resource": map[string]interface{}{
					"from": "HEADER",
					"key":  "X-Sentinel",
				},
				"circuitBreaker": map[string]interface{}{
					"rules": []interface{}{
						map[string]interface{}{
							"resource":               "cb1",
							"strategy":               "ERROR_COUNT",
							"retryTimeoutMs":         3000,
							"minRequestAmount":       2,
							"statIntervalMs":         2000,
							"threshold":              5,
							"probeNum":               2,
							"triggeredByStatusCodes": []uint32{500},
							"blockResponse": map[string]interface{}{
								"message":    "custom block resp: cb1",
								"statusCode": 503,
								"headers": map[string]string{
									"X-Sentinel-Blocked": "true",
								},
							},
						},
					},
				},
			}),
			run: func(t *testing.T) {
				hdr := http.Header{}
				hdr.Add("X-Sentinel", "cb1")

				isBreakerOpened := false
				// 10 requests, 5 of them will trigger the circuit breaker
				for i := 0; i < 10; i++ {
					resp, err := doGet(500, hdr, nil)
					assert.NoError(t, err)
					b, _ := io.ReadAll(resp.Body)
					if resp.StatusCode == 503 &&
						string(b) == "{\"msg\":\"custom block resp: cb1\"}" &&
						resp.Header.Get("X-Sentinel-Blocked") == "true" {
						isBreakerOpened = true
					}
				}
				assert.True(t, isBreakerOpened)

				// wait for the circuit breaker to be half-opened
				time.Sleep(3100 * time.Millisecond)

				for i := 0; i < 3; i++ {
					resp, err := doGet(200, hdr, nil)
					assert.NoError(t, err)
					assert.Equal(t, 200, resp.StatusCode)
					assert.Equal(t, "", resp.Header.Get("X-Sentinel-Blocked"))
				}
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

func TestSentinelMixture(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddBackendRoute(sentinelRoute),
	})
	if err != nil {
		t.Fatalf("failed to start dataplane: %v", err)
		return
	}
	defer dp.Stop()

	config := controlplane.NewSinglePluinConfig("sentinel", map[string]interface{}{
		"resource": map[string]interface{}{
			"from": "HEADER",
			"key":  "X-Sentinel",
		},
		"flow": map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{
					"resource":               "flow",
					"tokenCalculateStrategy": "DIRECT",
					"controlBehavior":        "REJECT",
					"threshold":              1,
					"statIntervalInMs":       1000,
					"blockResponse": map[string]interface{}{
						"message":    "custom block resp: flow",
						"statusCode": 503,
						"headers": map[string]string{
							"X-Sentinel-Type": "flow",
						},
					},
				},
			},
		},
		"hotSpot": map[string]interface{}{
			"params": []string{"123", "abc", "hello"},
			"rules": []interface{}{
				map[string]interface{}{
					"resource":        "hotspot",
					"metricType":      "QPS",
					"controlBehavior": "REJECT",
					"paramIndex":      0,
					"threshold":       1,
					"durationInSec":   1,
					"blockResponse": map[string]interface{}{
						"message":    "custom block resp: hotspot",
						"statusCode": 503,
						"headers": map[string]string{
							"X-Sentinel-Type": "hotspot",
						},
					},
				},
			},
		},
		"circuitBreaker": map[string]interface{}{
			"rules": []interface{}{
				map[string]interface{}{
					"resource":               "circuitbreaker",
					"strategy":               "ERROR_COUNT",
					"retryTimeoutMs":         3000,
					"minRequestAmount":       2,
					"statIntervalMs":         2000,
					"threshold":              5,
					"probeNum":               2,
					"triggeredByStatusCodes": []uint32{500},
					"blockResponse": map[string]interface{}{
						"message":    "custom block resp: circuitbreaker",
						"statusCode": 503,
						"headers": map[string]string{
							"X-Sentinel-Type": "circuitbreaker",
						},
					},
				},
			},
		},
	})

	controlPlane.UseGoPluginConfig(t, config, dp)

	// flow start
	hdr := http.Header{}
	hdr.Add("X-Sentinel", "flow")

	resp, _ := doGet(200, hdr, nil)
	assert.Equal(t, 200, resp.StatusCode)

	resp, _ = doGet(200, hdr, nil)
	assert.Equal(t, 503, resp.StatusCode)
	assert.Equal(t, "flow", resp.Header.Get("X-Sentinel-Type"))
	b, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "{\"msg\":\"custom block resp: flow\"}", string(b))

	time.Sleep(1100 * time.Millisecond)

	resp, _ = doGet(200, hdr, nil)
	assert.Equal(t, 200, resp.StatusCode)
	// flow end

	// hot spot start
	hdr = http.Header{}
	hdr.Add("X-Sentinel", "hotspot")

	resp, _ = doGet(200, hdr, nil)
	assert.Equal(t, 200, resp.StatusCode)

	resp, _ = doGet(200, hdr, nil)
	assert.Equal(t, 503, resp.StatusCode)
	assert.Equal(t, "hotspot", resp.Header.Get("X-Sentinel-Type"))
	b, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "{\"msg\":\"custom block resp: hotspot\"}", string(b))

	time.Sleep(1100 * time.Millisecond)

	resp, _ = doGet(200, hdr, nil)
	assert.Equal(t, 200, resp.StatusCode)
	// hot spot end

	// circuit breaker start
	hdr = http.Header{}
	hdr.Add("X-Sentinel", "circuitbreaker")

	isBreakerOpened := false
	// 10 requests, 5 of them will trigger the circuit breaker
	for i := 0; i < 10; i++ {
		resp, _ = doGet(500, hdr, nil)
		b, _ = io.ReadAll(resp.Body)
		if resp.StatusCode == 503 &&
			resp.Header.Get("X-Sentinel-Type") == "circuitbreaker" &&
			string(b) == "{\"msg\":\"custom block resp: circuitbreaker\"}" {
			isBreakerOpened = true
		}
	}
	assert.True(t, isBreakerOpened)

	// wait for the circuit breaker to be half-opened
	time.Sleep(3100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		resp, _ = doGet(200, hdr, nil)
		assert.Equal(t, 200, resp.StatusCode)
	}
	// circuit breaker end
}
