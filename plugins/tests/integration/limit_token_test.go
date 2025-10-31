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
	"bufio"
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
		Bootstrap: dataplane.Bootstrap().
			AddBackendRoute(limitTokenRoute).
			AddCluster(limitTokenCluster),
		NoErrorLogCheck: true,
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":6379", "redis")
	helper.WaitServiceUp(t, ":10903", "aicontentsecurity")

	config := controlplane.NewSinglePluginConfig("limittoken", map[string]interface{}{
		"rejected_code": 429,
		"rejected_msg":  "请求被限流",
		"rule": map[string]interface{}{
			"limit_by_header": "Authorization",
			"buckets": []map[string]interface{}{
				{"burst": 1000, "rate": 5, "round": 1},
			},
		},
		"redis": map[string]interface{}{
			"service_addr": "redis:6379",
		},
		"token_stats": map[string]interface{}{
			"window_size":        100,
			"min_samples":        5,
			"max_ratio":          4.0,
			"max_tokens_per_req": 200,
			"exceed_factor":      1.5,
		},
		"tokenizer":         "openai",
		"streaming_enabled": true,
		"gjson_config": map[string]interface{}{
			"request_content_path":            "messages",
			"request_model_path":              "model",
			"response_content_path":           "choices.0.message.content",
			"response_model_path":             "model",
			"response_completion_tokens_path": "usage.completion_tokens",
			"response_prompt_tokens_path":     "usage.prompt_tokens",
			"stream_response_content_path":    "choices.0.delta.content",
			"stream_response_model_path":      "model",
		},
	})

	controlPlane.UseGoPluginConfig(t, config, dp)
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", dp.Port())

	// ---- Non-streaming pass ----
	t.Run("Non-streaming pass", func(t *testing.T) {
		runNonStreamTest(t, client, url, "token-ns-pass", "hello,world!", http.StatusOK)
	})

	// ---- Non-streaming rejected ----
	t.Run("Non-streaming rejected", func(t *testing.T) {
		runNonStreamTest(t, client, url, "token-ns-reject", strings.Repeat("x", 10000), 429)
	})

	// ---- Streaming pass ----
	t.Run("Streaming pass", func(t *testing.T) {
		runStreamTest(t, client, url, "token-stream-pass", "hello,world!", false)
	})

	// ---- Streaming rejected ----
	t.Run("Streaming rejected", func(t *testing.T) {
		runStreamTest(t, client, url, "token-stream-reject", strings.Repeat("x", 10000), true)
	})
}

func runNonStreamTest(t *testing.T, client *http.Client, url, token, content string, expectCode int) {
	body := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
		"max_tokens": 5000,
		"stream":     false,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, expectCode, resp.StatusCode)
}

func runStreamTest(t *testing.T, client *http.Client, url, token string, content string, expectReject bool) {
	body := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
		"max_tokens": 5000,
		"stream":     true,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("read stream err: %v", err)
		}
		//all += line
		if strings.Contains(line, "[DONE]") {
			break
		}
	}

	if expectReject {
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	} else {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
}
