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

package limitToken

import (
	"github.com/tidwall/gjson"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/limitToken"
)

// TestFactory ensures the filter factory correctly creates a filter instance
func TestFactory(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	conf := &config{}

	// Create filter using factory
	f := factory(conf, cb).(*filter)

	// Assert that filter is properly initialized
	assert.NotNil(t, f)
	assert.Equal(t, cb, f.callbacks)
	assert.Equal(t, conf, f.config)
	assert.NotNil(t, f.sseParser)
	assert.NotNil(t, f.limiter)
}

// TestIsStream verifies the isStream helper for different content types
func TestIsStream(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{"SSE content type", "text/event-stream", true},
		{"Non-SSE content type", "application/json", false},
		{"Empty content type", "", false},
		{"Invalid content type", "invalid;type", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := envoy.NewRequestHeaderMap(http.Header{})
			if tt.contentType != "" {
				headers.Set("Content-Type", tt.contentType)
			}
			got := isStream(headers)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFilter simulates actual requests and responses through the filter
func TestFilter(t *testing.T) {
	conf := &config{
		CustomConfig: limitToken.CustomConfig{
			Config: limitToken.Config{
				RejectedCode: 409,               // HTTP status when request is limited
				RejectedMsg:  "Request limited", // Message when request is limited
				Rule: &limitToken.Rule{ // Limit rule by IP
					LimitBy: &limitToken.Rule_LimitByPerIp{},
					Buckets: []*limitToken.Bucket{
						{Burst: 100, Rate: 10, Round: 1},
					},
				},
				Redis: &limitToken.RedisConfig{ServiceAddr: "localhost:6379", Timeout: 5},
				TokenStats: &limitToken.TokenStatsConfig{
					WindowSize:      1000,
					MinSamples:      10,
					MaxRatio:        4.0,
					MaxTokensPerReq: 2000,
					ExceedFactor:    1.5,
				},
				Tokenizer: "openai",
				ExtractorConfig: &limitToken.Config_GjsonConfig{
					GjsonConfig: &limitToken.GjsonConfig{
						RequestContentPath:           "messages",
						RequestModelPath:             "model",
						ResponseContentPath:          "choices.0.message.content",
						ResponseModelPath:            "model",
						ResponseCompletionTokensPath: "usage.completion_tokens",
						ResponsePromptTokensPath:     "usage.prompt_tokens",
					},
				},
				StreamingEnabled: true,
			},
		},
	}

	// Initialize config
	err := conf.Init(envoy.NewFilterCallbackHandler())
	if err != nil {
		t.Fatal(err)
	}

	cb := envoy.NewFilterCallbackHandler()
	f := factory(conf, cb)

	// Test cases for request/response handling
	tests := []struct {
		name     string
		req      []byte
		resp     []byte
		wantCode int
		checkFn  func(t *testing.T, resp []byte)
	}{
		{
			name: "Valid single request",
			req: []byte(`{
				"model": "gpt-4o-mini",
				"messages": [{"role":"user","content":"Write a Go limiter middleware"}],
				"max_tokens": 50,
				"stream": false
			}`),
			resp: []byte(`{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"model": "gpt-4o-mini",
				"choices": [
					{"index":0,"message":{"role":"assistant","content":"This is the answer"},"finish_reason":"stop"}
				],
				"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}
			}`),
			wantCode: http.StatusOK,
			checkFn: func(t *testing.T, resp []byte) {
				assert.Equal(t, "This is the answer", gjson.GetBytes(resp, "choices.0.message.content").String())
				assert.Equal(t, int64(20), gjson.GetBytes(resp, "usage.completion_tokens").Int())
			},
		},
		{
			name: "Multiple choices request (n=2)",
			req: []byte(`{
				"model": "gpt-4o-mini",
				"messages": [{"role":"user","content":"Give me two implementations"}],
				"n": 2,
				"max_tokens": 50,
				"stream": false
			}`),
			resp: []byte(`{
				"id": "chatcmpl-456",
				"object": "chat.completion",
				"model": "gpt-4o-mini",
				"choices": [
					{"index":0,"message":{"role":"assistant","content":"Implementation 1"},"finish_reason":"stop"},
					{"index":1,"message":{"role":"assistant","content":"Implementation 2"},"finish_reason":"stop"}
				],
				"usage":{"prompt_tokens":15,"completion_tokens":40,"total_tokens":55}
			}`),
			wantCode: http.StatusOK,
			checkFn: func(t *testing.T, resp []byte) {
				assert.Equal(t, "Implementation 1", gjson.GetBytes(resp, "choices.0.message.content").String())
				assert.Equal(t, "Implementation 2", gjson.GetBytes(resp, "choices.1.message.content").String())
			},
		},
	}

	// Run each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			headers := envoy.NewRequestHeaderMap(h)
			reqBuf := envoy.NewBufferInstance(tt.req)
			rtMap := envoy.NewRequestTrailerMap(h)

			// Decode headers
			action := f.DecodeHeaders(headers, false)
			assert.Equal(t, api.Continue, action)

			// Process request body
			rhMap := envoy.NewResponseHeaderMap(h)
			action = f.DecodeRequest(headers, reqBuf, rtMap)
			assert.Equal(t, api.Continue, action)

			// Encode headers and response body
			action = f.EncodeHeaders(rhMap, true)
			respBuf := envoy.NewBufferInstance(tt.resp)
			action = f.EncodeData(respBuf, true)
			assert.Equal(t, api.Continue, action)

			// Optional checks
			if tt.checkFn != nil {
				tt.checkFn(t, tt.resp)
			}
		})
	}
}
