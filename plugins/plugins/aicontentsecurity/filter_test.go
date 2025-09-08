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

package aicontentsecurity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/extractor"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	plugintype "mosn.io/htnn/types/plugins/aicontentsecurity"
)

// Mock implementations
type mockModerator struct {
	requestFunc  func(ctx context.Context, content string, idMap map[string]string) (*moderation.Result, error)
	responseFunc func(ctx context.Context, content string, idMap map[string]string) (*moderation.Result, error)
}

func (m *mockModerator) Request(ctx context.Context, content string, idMap map[string]string) (*moderation.Result, error) {
	if m.requestFunc != nil {
		return m.requestFunc(ctx, content, idMap)
	}
	return &moderation.Result{Allow: true}, nil
}

func (m *mockModerator) Response(ctx context.Context, content string, idMap map[string]string) (*moderation.Result, error) {
	if m.responseFunc != nil {
		return m.responseFunc(ctx, content, idMap)
	}
	return &moderation.Result{Allow: true}, nil
}

type mockExtractor struct {
	setDataFunc               func(data []byte) error
	requestContentFunc        func() string
	responseContentFunc       func() string
	streamResponseContentFunc func() string
	idsFromRequestDataFunc    func(idMap map[string]string)
	idsFromRequestHeadersFunc func(headers api.RequestHeaderMap, idMap map[string]string)
}

func (m *mockExtractor) SetData(data []byte) error {
	if m.setDataFunc != nil {
		return m.setDataFunc(data)
	}
	return nil
}

func (m *mockExtractor) RequestContent() string {
	if m.requestContentFunc != nil {
		return m.requestContentFunc()
	}
	return ""
}

func (m *mockExtractor) ResponseContent() string {
	if m.responseContentFunc != nil {
		return m.responseContentFunc()
	}
	return ""
}

func (m *mockExtractor) StreamResponseContent() string {
	if m.streamResponseContentFunc != nil {
		return m.streamResponseContentFunc()
	}
	return ""
}

func (m *mockExtractor) IDsFromRequestData(idMap map[string]string) {
	if m.idsFromRequestDataFunc != nil {
		m.idsFromRequestDataFunc(idMap)
	}
}

func (m *mockExtractor) IDsFromRequestHeaders(headers api.RequestHeaderMap, idMap map[string]string) {
	if m.idsFromRequestHeadersFunc != nil {
		m.idsFromRequestHeadersFunc(headers, idMap)
	}
}

type testConfig struct {
	streamingEnabled   bool
	charLimit          int
	overlapLength      int
	requestContentPath string
	respContentPath    string
	streamRespPath     string
	moderator          moderation.Moderator
	extractor          extractor.Extractor
}

func getCfg(opts testConfig) *config {
	if opts.charLimit == 0 {
		opts.charLimit = 10
	}
	if opts.requestContentPath == "" {
		opts.requestContentPath = "text"
	}
	if opts.respContentPath == "" {
		opts.respContentPath = "response_text"
	}
	if opts.streamRespPath == "" {
		opts.streamRespPath = "choices.0.delta.content"
	}

	conf := &config{}
	conf.CustomConfig.Config = plugintype.Config{
		ModerationCharLimit:          int64(opts.charLimit),
		ModerationChunkOverlapLength: int64(opts.overlapLength),
		StreamingEnabled:             opts.streamingEnabled,
		ProviderConfig: &plugintype.Config_LocalModerationServiceConfig{
			LocalModerationServiceConfig: &plugintype.LocalModerationServiceConfig{
				BaseUrl: "http://mock.test.service",
			},
		},
		ExtractorConfig: &plugintype.Config_GjsonConfig{
			GjsonConfig: &plugintype.GjsonConfig{
				RequestContentPath:        opts.requestContentPath,
				ResponseContentPath:       opts.respContentPath,
				StreamResponseContentPath: opts.streamRespPath,
			},
		},
	}

	// Use custom moderator and extractor if provided, otherwise use defaults
	if opts.moderator != nil {
		conf.moderator = opts.moderator
	} else {
		conf.moderator = &mockModerator{}
	}

	if opts.extractor != nil {
		conf.extractor = opts.extractor
	} else {
		conf.extractor = &mockExtractor{}
	}

	// Set default moderation timeout
	conf.moderationTimeout = 3 * time.Second

	return conf
} // TestDecode covers all request-side (Decode) filter behaviors.
func TestDecode(t *testing.T) {
	t.Run("Headers", func(t *testing.T) {
		var extractorCalled bool
		mockExt := &mockExtractor{
			idsFromRequestHeadersFunc: func(api.RequestHeaderMap, map[string]string) {
				extractorCalled = true
			},
		}

		cb := envoy.NewFilterCallbackHandler()
		cfg := getCfg(testConfig{extractor: mockExt})
		f := factory(cfg, cb).(*filter)
		h := http.Header{}
		h.Set("Content-Type", "application/json")
		h.Set("session_id", "session123")
		headers := envoy.NewRequestHeaderMap(h)

		result := f.DecodeHeaders(headers, false)

		assert.Equal(t, api.Continue, result)
		assert.True(t, extractorCalled, "IDsFromRequestHeaders should have been called")
	})

	t.Run("Non-Stream", func(t *testing.T) {
		t.Run("SingleChunk_SafeContent", func(t *testing.T) {
			mockMod := &mockModerator{
				requestFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return &moderation.Result{Allow: true}, nil
				},
			}
			mockExt := &mockExtractor{
				requestContentFunc: func() string {
					return "this is a safe message"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.DecodeHeaders(envoy.NewRequestHeaderMap(h), false)

			body := `{"text": "this is a safe message", "session_id": "s1"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.DecodeData(buf, true)

			assert.Equal(t, api.Continue, result)
			assert.Equal(t, body, string(buf.Bytes()))
		})

		t.Run("SingleChunk_UnsafeContent", func(t *testing.T) {
			unsafeResult := &moderation.Result{Allow: false, Reason: "blocked by security"}
			mockMod := &mockModerator{
				requestFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return unsafeResult, nil
				},
			}
			mockExt := &mockExtractor{
				requestContentFunc: func() string {
					return "this is a dangerous message"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.DecodeHeaders(envoy.NewRequestHeaderMap(h), false)

			body := `{"text": "this is a dangerous message", "session_id": "s2"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.DecodeData(buf, true)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok, "result should be a LocalResponse")
			assert.Equal(t, http.StatusBadGateway, res.Code)
			assert.Equal(t, "blocked by security", res.Msg)
		})

		t.Run("MultiChunk_SafeContent", func(t *testing.T) {
			var moderatedContent string
			mockMod := &mockModerator{
				requestFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					moderatedContent = content
					return &moderation.Result{Allow: true}, nil
				},
			}
			mockExt := &mockExtractor{
				requestContentFunc: func() string {
					return "this is a safe message"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{charLimit: 100, moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.DecodeHeaders(envoy.NewRequestHeaderMap(h), false)

			part1 := `{"text": "this is a `
			buf1 := envoy.NewBufferInstance([]byte(part1))
			result1 := f.DecodeData(buf1, false)
			assert.Equal(t, api.Continue, result1)
			assert.Empty(t, buf1.Bytes(), "Buffer for part 1 should be drained")

			part2 := `safe message", "session_id": "s4"}`
			buf2 := envoy.NewBufferInstance([]byte(part2))
			result2 := f.DecodeData(buf2, true)
			assert.Equal(t, api.Continue, result2)

			assert.Equal(t, "this is a safe message", moderatedContent)
			assert.Equal(t, part1+part2, string(buf2.Bytes()))
		})

		t.Run("ModeratorError", func(t *testing.T) {
			moderatorErr := errors.New("moderation service unavailable")
			mockMod := &mockModerator{
				requestFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return nil, moderatorErr
				},
			}
			mockExt := &mockExtractor{
				requestContentFunc: func() string {
					return "any message"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.DecodeHeaders(envoy.NewRequestHeaderMap(h), false)

			body := `{"text": "any message", "session_id": "s3"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.DecodeData(buf, true)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok, "result should be a LocalResponse")
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("ExtractorError", func(t *testing.T) {
			mockExt := &mockExtractor{
				setDataFunc: func(data []byte) error {
					return errors.New("invalid json")
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			body := `{"text": "this is not valid json`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.DecodeData(buf, true)
			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("NoContentField", func(t *testing.T) {
			var moderatorCalled bool
			mockMod := &mockModerator{
				requestFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					moderatorCalled = true
					return &moderation.Result{Allow: true}, nil
				},
			}
			mockExt := &mockExtractor{
				requestContentFunc: func() string {
					return "" // No content extracted
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.DecodeHeaders(envoy.NewRequestHeaderMap(h), false)

			body := `{"some_other_field": "some value", "session_id": "s5"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.DecodeData(buf, true)

			assert.Equal(t, api.Continue, result)
			assert.False(t, moderatorCalled, "Request should not have been called as no content was extracted")
		})
	})
}

// TestEncode covers all response-side (Encode) filter behaviors.
func TestEncode(t *testing.T) {
	t.Run("Non-Stream", func(t *testing.T) {
		t.Run("Headers", func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			f := factory(getCfg(testConfig{}), cb).(*filter)
			h := http.Header{}
			h.Set("Content-Type", "application/json")
			headers := envoy.NewResponseHeaderMap(h)

			result := f.EncodeHeaders(headers, false)

			assert.Equal(t, api.Continue, result)
			assert.False(t, f.streamResponse, "streamResponse should be false for non-stream responses")
		})

		t.Run("SafeContent", func(t *testing.T) {
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return &moderation.Result{Allow: true}, nil
				},
			}
			mockExt := &mockExtractor{
				responseContentFunc: func() string {
					return "this is a safe response"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			body := `{"response_text": "this is a safe response"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.EncodeData(buf, true)

			assert.Equal(t, api.Continue, result)
			assert.Equal(t, body, string(buf.Bytes()))
		})

		t.Run("UnsafeContent", func(t *testing.T) {
			unsafeResult := &moderation.Result{Allow: false, Reason: "unsafe response"}
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return unsafeResult, nil
				},
			}
			mockExt := &mockExtractor{
				responseContentFunc: func() string {
					return "this is a bad response"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			body := `{"response_text": "this is a bad response"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.EncodeData(buf, true)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
			assert.Equal(t, "unsafe response", res.Msg)
		})

		t.Run("ModeratorError", func(t *testing.T) {
			moderatorErr := errors.New("moderation service unavailable")
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return nil, moderatorErr
				},
			}
			mockExt := &mockExtractor{
				responseContentFunc: func() string {
					return "any response"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			body := `{"response_text": "any response"}`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.EncodeData(buf, true)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("MultiPartBody_SafeContent", func(t *testing.T) {
			var moderatedContent string
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					moderatedContent = content
					return &moderation.Result{Allow: true}, nil
				},
			}
			mockExt := &mockExtractor{
				responseContentFunc: func() string {
					return "this is a multi-part safe response"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{charLimit: 100, moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			part1 := `{"response_text": "this is `
			buf1 := envoy.NewBufferInstance([]byte(part1))
			result1 := f.EncodeData(buf1, false)
			assert.Equal(t, api.Continue, result1)
			assert.Empty(t, buf1.Bytes())

			part2 := `a multi-part safe response"}`
			buf2 := envoy.NewBufferInstance([]byte(part2))
			result2 := f.EncodeData(buf2, true)
			assert.Equal(t, api.Continue, result2)

			assert.Equal(t, "this is a multi-part safe response", moderatedContent)
			assert.Equal(t, fmt.Sprintf("%s%s", part1, part2), string(buf2.Bytes()))
		})

		t.Run("MultiPartBody_SetDataError", func(t *testing.T) {
			cfg := getCfg(testConfig{charLimit: 100})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", &moderation.Result{Allow: true}, nil)
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "application/json")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			part1 := `{"response_text": "this is a safe response"}`
			buf1 := envoy.NewBufferInstance([]byte(part1))
			f.EncodeData(buf1, false)

			buf2 := envoy.NewBufferInstance(nil)
			setDataErr := errors.New("failed to set data")
			patches.ApplyMethodReturn(buf2, "Set", setDataErr)

			result := f.EncodeData(buf2, true)
			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})
	})

	t.Run("Stream", func(t *testing.T) {
		t.Run("Headers_IsStream", func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			f := factory(getCfg(testConfig{}), cb).(*filter)
			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			headers := envoy.NewResponseHeaderMap(h)

			result := f.EncodeHeaders(headers, false)

			assert.Equal(t, api.WaitAllData, result)
		})

		t.Run("Headers_NoContentType", func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			f := factory(getCfg(testConfig{}), cb).(*filter)
			h := http.Header{}
			headers := envoy.NewResponseHeaderMap(h)

			result := f.EncodeHeaders(headers, false)

			assert.Equal(t, api.Continue, result)
			assert.False(t, f.streamResponse, "streamResponse should be false when Content-Type is missing")
		})

		t.Run("SplitEventAcrossFrames_Safe", func(t *testing.T) {
			var moderatedContent string
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					moderatedContent = content
					return &moderation.Result{Allow: true}, nil
				},
			}
			mockExt := &mockExtractor{
				streamResponseContentFunc: func() string {
					return "Part 1 of the message."
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{streamingEnabled: true, charLimit: 22, moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			part1 := `data: {"choices":[{"delta":{"content":"Part 1 `
			buf1 := envoy.NewBufferInstance([]byte(part1))
			result1 := f.EncodeData(buf1, false)
			assert.Equal(t, api.Continue, result1)
			assert.Empty(t, buf1.Bytes())

			part2 := `of the message."}}]}` + "\n\n"
			buf2 := envoy.NewBufferInstance([]byte(part2))
			result2 := f.EncodeData(buf2, false)
			assert.Equal(t, api.Continue, result2)
			assert.Equal(t, part1+part2, string(buf2.Bytes()))

			assert.Equal(t, "Part 1 of the message.", moderatedContent)
		})

		t.Run("StreamingDisabled_BuffersUntilEnd", func(t *testing.T) {
			var moderatedContent string
			var callCount int
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					callCount++
					if callCount == 1 {
						moderatedContent = content
					}
					return &moderation.Result{Allow: true}, nil
				},
			}
			var extractorCallCount int
			mockExt := &mockExtractor{
				streamResponseContentFunc: func() string {
					extractorCallCount++
					if extractorCallCount == 1 {
						return "Hello world. This is a test."
					}
					return ""
				},
			}

			// Test case where streaming is disabled in config, so all events are buffered
			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{streamingEnabled: false, charLimit: 100, moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			hdr := envoy.NewResponseHeaderMap(h)
			f.EncodeHeaders(hdr, false)

			sseEvent1 := `data: {"choices":[{"delta":{"content":"Hello world. "}}]}` + "\n\n"
			sseEvent2 := `data: {"choices":[{"delta":{"content":"This is a test."}}]}` + "\n\n"

			buf := envoy.NewBufferInstance([]byte(sseEvent1 + sseEvent2))
			result := f.EncodeResponse(hdr, buf, nil)
			assert.Equal(t, api.Continue, result)

			// Check that the full content was moderated at once
			assert.Equal(t, "Hello world. This is a test.", moderatedContent)
			// Check that all buffered events are written back at the end
			assert.Equal(t, sseEvent1+sseEvent2, string(buf.Bytes()))
		})

		// Simplified test cases for complex scenarios
		t.Run("UnsafeContent_BlocksStream", func(t *testing.T) {
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return &moderation.Result{Allow: false, Reason: "blocked content"}, nil
				},
			}
			mockExt := &mockExtractor{
				streamResponseContentFunc: func() string {
					return "unsafe content"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{streamingEnabled: true, moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent := `data: {"choices":[{"delta":{"content":"unsafe content"}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))
			result := f.EncodeData(buf, false)

			assert.Equal(t, api.Continue, result)
			// The filter should have set an error message and marked the stream as closed
			assert.True(t, f.streamCloseFlag, "streamCloseFlag should be set after blocking")
			expectedError := "event: error\ndata: blocked content\n\n"
			assert.Equal(t, expectedError, string(buf.Bytes()))
		})

		t.Run("ModeratorError_ReturnsGatewayError", func(t *testing.T) {
			mockMod := &mockModerator{
				responseFunc: func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
					return nil, errors.New("moderation service error")
				},
			}
			mockExt := &mockExtractor{
				streamResponseContentFunc: func() string {
					return "some content"
				},
			}

			cb := envoy.NewFilterCallbackHandler()
			cfg := getCfg(testConfig{streamingEnabled: true, moderator: mockMod, extractor: mockExt})
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent := `data: {"choices":[{"delta":{"content":"some content"}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))
			result := f.EncodeData(buf, true)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})
	})
}
