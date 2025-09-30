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
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

var (
	//go:embed testdata/ai_mock_service_route.yml
	limitTokenRoute string

	//go:embed testdata/ai_mock_service_cluster.yml
	limitTokenCluster string
)

func TestLimitToken(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddBackendRoute(limitTokenRoute).AddCluster(limitTokenCluster),
	})

	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":10901", "aimockservices")

	config := controlplane.NewSinglePluginConfig("limitToken", map[string]interface{}{
		"rejected_code": 429,
		"rejected_msg":  "请求被限流",
		"rule": map[string]interface{}{
			"limit_by_header": "Authorization",
			"buckets": []map[string]interface{}{
				{
					"burst": 10,
					"rate":  5,
					"round": 1,
				},
			},
		},
		"redis": map[string]interface{}{
			"service_addr": "localhost:6379",
		},
		"token_stats": map[string]interface{}{
			"window_size":        100,
			"min_samples":        5,
			"max_ratio":          4.0,
			"max_tokens_per_req": 200,
			"exceed_factor":      1.5,
		},
		"tokenizer": "openai",
		"extractor_config": map[string]interface{}{
			"request_content_path":         "messages.0.content",
			"request_model_path":           "model",
			"response_content_path":        "choices.0.message.content",
			"response_model_path":          "choices.0.message.model",
			"stream_response_content_path": "choices.0.delta.content",
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", dp.Port())

	t.Run("Non-streaming pass", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": "Hello, world!"},
			},
			"model":  "gpt-3.5-turbo",
			"stream": false,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "token-1")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Non-streaming rejected", func(t *testing.T) {
		// 模拟请求超大 token
		requestBody := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": strings.Repeat("x", 10000)},
			},
			"model":  "gpt-3.5-turbo",
			"stream": false,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "token-1")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, 429, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "请求被限流")
	})

	t.Run("Streaming pass", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": "streaming message"},
			},
			"model":     "gpt-3.5-turbo",
			"stream":    true,
			"event_num": 3,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "token-2")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		allData, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(allData), "data:")
	})

	t.Run("Streaming rejected", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": strings.Repeat("y", 10000)},
			},
			"model":  "gpt-3.5-turbo",
			"stream": true,
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "token-2")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "请求被限流")
	})
}
