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
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	plugintype "mosn.io/htnn/types/plugins/aicontentsecurity"
)

type testConfig struct {
	streamingEnabled   bool
	charLimit          int
	overlapLength      int
	requestContentPath string
	respContentPath    string
	streamRespPath     string
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
	err := conf.Init(nil)
	if err != nil {
		panic(fmt.Sprintf("config init error: %v", err))
	}
	return conf
}

// TestDecode covers all request-side (Decode) filter behaviors.
func TestDecode(t *testing.T) {
	t.Run("Headers", func(t *testing.T) {
		cb := envoy.NewFilterCallbackHandler()
		f := factory(getCfg(testConfig{}), cb).(*filter)
		h := http.Header{}
		h.Set("Content-Type", "application/json")
		h.Set("session_id", "session123")
		headers := envoy.NewRequestHeaderMap(h)

		var extractorCalled bool
		patches := gomonkey.ApplyMethodFunc(f.config.extractor, "IDsFromRequestHeaders", func(api.RequestHeaderMap, map[string]string) {
			extractorCalled = true
		})
		defer patches.Reset()

		result := f.DecodeHeaders(headers, false)

		assert.Equal(t, api.Continue, result)
		assert.True(t, extractorCalled, "IDsFromRequestHeaders should have been called")
	})

	t.Run("Non-Stream", func(t *testing.T) {
		t.Run("SingleChunk_SafeContent", func(t *testing.T) {
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Request", &moderation.Result{Allow: true}, nil)
			defer patches.Reset()

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
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			unsafeResult := &moderation.Result{Allow: false, Reason: "blocked by security"}
			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Request", unsafeResult, nil)
			defer patches.Reset()

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
			cfg := getCfg(testConfig{charLimit: 100})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderatedContent string
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Request", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				moderatedContent = content
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

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
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			moderatorErr := errors.New("moderation service unavailable")
			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Request", nil, moderatorErr)
			defer patches.Reset()

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
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			body := `{"text": "this is not valid json`
			buf := envoy.NewBufferInstance([]byte(body))

			result := f.DecodeData(buf, true)
			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("NoContentField", func(t *testing.T) {
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderatorCalled bool
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Request", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				moderatorCalled = true
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

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
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", &moderation.Result{Allow: true}, nil)
			defer patches.Reset()

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
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			unsafeResult := &moderation.Result{Allow: false, Reason: "unsafe response"}
			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", unsafeResult, nil)
			defer patches.Reset()

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
			cfg := getCfg(testConfig{})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			moderatorErr := errors.New("moderation service unavailable")
			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", nil, moderatorErr)
			defer patches.Reset()

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
			cfg := getCfg(testConfig{charLimit: 100})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderatedContent string
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				moderatedContent = content
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

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

			assert.Equal(t, api.Continue, result)
			assert.True(t, f.streamResponse, "streamResponse should be true for stream responses")
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
			cfg := getCfg(testConfig{streamingEnabled: true, charLimit: 22})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderatedContent string
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				moderatedContent = content
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

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

		t.Run("SingleEventMultiChunk_Safe_FlushAtEnd", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true, charLimit: 20, overlapLength: 6})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderationCalls []string
			var mu sync.Mutex
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				mu.Lock()
				moderationCalls = append(moderationCalls, content)
				mu.Unlock()
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent := `data: {"choices":[{"delta":{"content":"This is a long test message for streaming."}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))
			result := f.EncodeData(buf, false)

			assert.Equal(t, api.Continue, result)
			assert.Empty(t, buf.Bytes())

			endStreamBuf := envoy.NewBufferInstance(nil)
			result = f.EncodeData(endStreamBuf, true)
			assert.Equal(t, api.Continue, result)

			require.Len(t, moderationCalls, 3)
			assert.Contains(t, moderationCalls, "This is a long test ")
			assert.Contains(t, moderationCalls, " test message for st")
			assert.Contains(t, moderationCalls, "for streaming.")

			assert.Equal(t, sseEvent, string(endStreamBuf.Bytes()))
		})

		t.Run("MultiEvents_EventReleaseLogic_Safe", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true, charLimit: 25, overlapLength: 5})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderationCalls []string
			var mu sync.Mutex
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				mu.Lock()
				moderationCalls = append(moderationCalls, content)
				mu.Unlock()
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent1 := `data: {"choices":[{"delta":{"content":"Hello world."}}]}` + "\n\n"
			sseEvent2 := `data: {"choices":[{"delta":{"content":"This is a test message."}}]}` + "\n\n"
			sseEvent3 := `data: {"choices":[{"delta":{"content":"Another one."}}]}` + "\n\n"

			buf1 := envoy.NewBufferInstance([]byte(sseEvent1 + sseEvent2))
			result1 := f.EncodeData(buf1, false)
			assert.Equal(t, api.Continue, result1)
			assert.Equal(t, sseEvent1, string(buf1.Bytes()))

			require.Len(t, moderationCalls, 1)
			assert.Equal(t, "Hello world.This is a tes", moderationCalls[0])

			buf2 := envoy.NewBufferInstance([]byte(sseEvent3))
			result2 := f.EncodeData(buf2, true)
			assert.Equal(t, api.Continue, result2)
			assert.Equal(t, sseEvent2+sseEvent3, string(buf2.Bytes()))

			require.Len(t, moderationCalls, 3)
			assert.Contains(t, moderationCalls, "a test message.Another on")
			assert.Contains(t, moderationCalls, "er one.")
		})

		t.Run("UnsafeContentInSecondChunk", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true, charLimit: 20})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)
			failMsg := "unsafe part 2"

			var mu sync.Mutex
			callCount := 0
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				mu.Lock()
				callCount++
				currentCallCount := callCount
				mu.Unlock()

				if currentCallCount == 1 {
					return &moderation.Result{Allow: true}, nil
				}
				return &moderation.Result{Allow: false, Reason: failMsg}, nil
			})
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent := `data: {"choices":[{"delta":{"content":"This is a safe first part, but the second part is not."}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))
			result := f.EncodeData(buf, true)
			assert.Equal(t, api.Continue, result)
			assert.Equal(t, fmt.Sprintf("event: error\ndata: %s\n\n", failMsg), buf.String())
		})

		t.Run("BlockedStatePropagation", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", &moderation.Result{Allow: false, Reason: "blocked"}, nil)
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			// First event is unsafe, triggers blocked state
			sseEvent1 := `data: {"choices":[{"delta":{"content":"bad content, bad content"}}]}` + "\n\n"
			buf1 := envoy.NewBufferInstance([]byte(sseEvent1))
			result1 := f.EncodeData(buf1, false)
			assert.Equal(t, api.Continue, result1)
			assert.Equal(t, "event: error\ndata: blocked\n\n", string(buf1.Bytes()))
			assert.True(t, f.streamCloseFlag, "streamCloseFlag should be set to true after blocking")

			// Second event should be completely dropped
			sseEvent2 := `data: {"choices":[{"delta":{"content":"more content"}}]}` + "\n\n"
			buf2 := envoy.NewBufferInstance([]byte(sseEvent2))
			result2 := f.EncodeData(buf2, false)
			assert.Empty(t, buf2.Bytes(), "Buffer should be empty as the stream is closed")
			res, ok := result2.(*api.LocalResponse)
			require.True(t, ok, "A LocalResponse should be returned for subsequent data when stream is closed")
			assert.Equal(t, http.StatusBadGateway, res.Code)

			// End of stream should also be handled correctly
			buf3 := envoy.NewBufferInstance(nil)
			result3 := f.EncodeData(buf3, true)
			res, ok = result3.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("SSEParserError_Patched", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			parserErr := errors.New("forced parser error")
			patches := gomonkey.ApplyMethodReturn(f.sseParser, "TryParse", nil, parserErr)
			defer patches.Reset()

			validEvent := `data: {"choices":[{"delta":{"content":"this is a valid event"}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(validEvent))
			result := f.EncodeData(buf, false)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("StreamSetErrorPayloadError", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", &moderation.Result{Allow: false, Reason: "blocked"}, nil)
			defer patches.Reset()

			sseEvent := `data: {"choices":[{"delta":{"content":"unsafe content"}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))

			setErr := errors.New("buffer set failed")
			patches.ApplyMethodReturn(buf, "Set", setErr)

			result := f.EncodeData(buf, false)
			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("StreamAppendParsedBytesError", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", &moderation.Result{Allow: true}, nil)
			defer patches.Reset()

			sseEvent := `data: {"choices":[{"delta":{"content":"safe content"}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))

			appendErr := errors.New("buffer append failed")
			patches.ApplyMethodReturn(buf, "Append", appendErr)

			result := f.EncodeData(buf, false)
			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("StreamAppendUnparsedBytesError", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			patches := gomonkey.ApplyMethodReturn(f.config.moderator, "Response", &moderation.Result{Allow: true}, nil)
			defer patches.Reset()

			sseEvent := `data: {"choices":[{"delta":{"content":"safe content"}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))

			appendErr := errors.New("buffer append failed")
			var callCount int
			patches.ApplyMethodFunc(buf, "Append", func([]byte) error {
				callCount++
				if callCount > 1 {
					return appendErr
				}
				return nil
			})

			result := f.EncodeData(buf, true)
			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})

		t.Run("StreamingDisabled_BuffersUntilEnd", func(t *testing.T) {
			// Test case where streaming is disabled in config, so all events are buffered
			cfg := getCfg(testConfig{streamingEnabled: false, charLimit: 100})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)

			var moderatedContent string
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				moderatedContent = content
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent1 := `data: {"choices":[{"delta":{"content":"Hello world. "}}]}` + "\n\n"
			sseEvent2 := `data: {"choices":[{"delta":{"content":"This is a test."}}]}` + "\n\n"

			buf1 := envoy.NewBufferInstance([]byte(sseEvent1))
			result1 := f.EncodeData(buf1, false)
			assert.Equal(t, api.Continue, result1)
			assert.Empty(t, buf1.Bytes(), "Buffer should be drained as event is cached")

			buf2 := envoy.NewBufferInstance([]byte(sseEvent2))
			result2 := f.EncodeData(buf2, false)
			assert.Equal(t, api.Continue, result2)
			assert.Empty(t, buf2.Bytes(), "Buffer should be drained as event is cached")

			// Now end the stream, which should trigger the moderation
			endBuf := envoy.NewBufferInstance(nil)
			resultEnd := f.EncodeData(endBuf, true)
			assert.Equal(t, api.Continue, resultEnd)

			// Check that the full content was moderated at once
			assert.Equal(t, "Hello world. This is a test.", moderatedContent)
			// Check that all buffered events are written back at the end
			assert.Equal(t, sseEvent1+sseEvent2, string(endBuf.Bytes()))
		})

		t.Run("ConcurrencyModerationError", func(t *testing.T) {
			cfg := getCfg(testConfig{streamingEnabled: true, charLimit: 10})
			cb := envoy.NewFilterCallbackHandler()
			f := factory(cfg, cb).(*filter)
			moderatorErr := errors.New("moderation service unavailable")

			var mu sync.Mutex
			callCount := 0
			patches := gomonkey.ApplyMethodFunc(f.config.moderator, "Response", func(_ context.Context, content string, _ map[string]string) (*moderation.Result, error) {
				mu.Lock()
				callCount++
				currentCallCount := callCount
				mu.Unlock()

				if currentCallCount == 2 {
					return nil, moderatorErr
				}
				return &moderation.Result{Allow: true}, nil
			})
			defer patches.Reset()

			h := http.Header{}
			h.Set("Content-Type", "text/event-stream")
			f.EncodeHeaders(envoy.NewResponseHeaderMap(h), false)

			sseEvent := `data: {"choices":[{"delta":{"content":"This content will trigger multiple moderation calls."}}]}` + "\n\n"
			buf := envoy.NewBufferInstance([]byte(sseEvent))
			result := f.EncodeData(buf, true)

			res, ok := result.(*api.LocalResponse)
			require.True(t, ok)
			assert.Equal(t, http.StatusBadGateway, res.Code)
		})
	})
}
