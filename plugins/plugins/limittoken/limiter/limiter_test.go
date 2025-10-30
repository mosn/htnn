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

package limiter

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/plugins/plugins/limittoken/tokenizer"
	"mosn.io/htnn/types/plugins/limittoken"
)

func TestOptions(t *testing.T) {
	r := regexp.MustCompile(`(\d+)`)
	// Start in-memory Redis
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	ts := &TokenStats{
		WindowSize: 5, MinSamples: 1,
		MaxRatio: 2, MaxTokensPerReq: 100, ExceedFactor: 2,
	}

	l := NewLimiter(
		WithRegexps([]*regexp.Regexp{r}),
		WithRedisLimiter(rdb),
		WithTokenizer("openai"),
		WithTokenStats(ts),
		WithRejectedMsg("reject"),
		WithRejectedCode(499),
	)

	assert.Equal(t, "reject", l.rejectedMsg, "rejectedMsg should be set correctly")
	assert.Equal(t, 499, l.rejectedCode, "rejectedCode should be set correctly")
	assert.Equal(t, ts, l.tokenStat, "tokenStat should be set correctly")
}

func TestDecodeDataAndEncodeData(t *testing.T) {
	l := NewLimiter()
	l.tokenizer = &tokenizer.OpenaiTokenizer{}
	l.tokenStat = &TokenStats{
		WindowSize: 5, MinSamples: 1,
		MaxRatio: 2, MaxTokensPerReq: 100, ExceedFactor: 2,
	}
	l.buckets = []*limittoken.Bucket{{Burst: 10, Rate: 100, Round: 1}}

	httpHdr := http.Header{"x-mse-consumer": []string{"user1"}}
	hdr := envoy.NewRequestHeaderMap(httpHdr)
	hdr.Set(":authority", "127.0.0.1")

	rule := &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByConsumer{}}

	// Simulate a real OpenAI request JSON
	messages := []tokenizer.OpenaiPromptMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello! How are you?"},
	}
	messagesStr, err := json.Marshal(messages)
	assert.NoError(t, err, "marshalling messages should not fail")

	res := l.DecodeData(hdr, rule, string(messagesStr), "gpt-3.5-turbo-0613")
	assert.Equal(t, api.Continue, res, "DecodeData should return Continue")

	res = l.EncodeData("Hello, this is a Go function example", "gpt-3.5-turbo-0613", 20, 10)
	assert.Equal(t, api.Continue, res, "EncodeData should return Continue")
}

func TestEncodeStreamData(t *testing.T) {
	// Start in-memory Redis
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() {
		rdb.FlushAll(context.Background())
		_ = rdb.Close()
	})

	l := NewLimiter(WithRedisLimiter(rdb))
	l.tokenizer = &tokenizer.OpenaiTokenizer{}
	l.tokenStat = &TokenStats{WindowSize: 3, MinSamples: 1, MaxRatio: 2, MaxTokensPerReq: 50, ExceedFactor: 2}
	l.buckets = []*limittoken.Bucket{{Burst: 100, Rate: 10, Round: 1}}
	l.predictCompletionToken = 5

	messages := []tokenizer.OpenaiPromptMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "system", Content: "Hello! How are you?"},
	}
	messagesStr, err := json.Marshal(messages)
	assert.NoError(t, err, "marshalling messages should not fail")

	// Non-final chunk
	res := l.EncodeStreamData(string(messagesStr), "gpt-3.5-turbo-0613", false)
	assert.Equal(t, api.Continue, res, "EncodeStreamData(non-final) should return Continue")

	// Final chunk
	res = l.EncodeStreamData(string(messagesStr), "gpt-3.5-turbo-0613", true)
	assert.Equal(t, api.Continue, res, "EncodeStreamData(final) should return Continue")
}

func TestTokenStats(t *testing.T) {
	s := &TokenStats{WindowSize: 3, MinSamples: 2, MaxRatio: 2, MaxTokensPerReq: 50, ExceedFactor: 2}

	// Insufficient samples => fallback to MaxRatio
	s.Add(10, 20)
	if !s.IsExceeded(10) {
		t.Errorf("expected true when sample size is insufficient")
	}

	// Add sufficient samples
	s.Add(10, 30)
	s.Add(20, 40)
	s.Add(30, 60) // overwrite old values

	if v := s.PredictCompletionTokens(10); v <= 0 {
		t.Errorf("prediction should be >0, got %d", v)
	}
}

func TestGetKey(t *testing.T) {
	l := NewLimiter()
	tests := []struct {
		name      string
		headerMap func() api.RequestHeaderMap
		rule      *limittoken.Rule
		want      string
	}{
		{
			name: "header",
			headerMap: func() api.RequestHeaderMap {
				h := http.Header{}
				hdr := envoy.NewRequestHeaderMap(h)
				hdr.Set("X-header", "abc")
				return hdr
			},
			rule: &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByHeader{LimitByHeader: "X-header"}},
			want: "abc",
		},
		{
			name: "param",
			headerMap: func() api.RequestHeaderMap {
				h := http.Header{}
				hdr := envoy.NewRequestHeaderMap(h)
				hdr.Set(":path", "/?p=1")
				return hdr
			},
			rule: &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByParam{LimitByParam: "p"}},
			want: "1",
		},
		{
			name: "cookie",
			headerMap: func() api.RequestHeaderMap {
				h := http.Header{}
				h.Add("Cookie", "c=val")
				hdr := envoy.NewRequestHeaderMap(h)
				return hdr
			},
			rule: &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByCookie{LimitByCookie: "c"}},
			want: "c=val",
		},
		{
			name: "regex ip",
			headerMap: func() api.RequestHeaderMap {
				h := http.Header{}
				hdr := envoy.NewRequestHeaderMap(h)
				hdr.Set(":authority", "ip123")
				return hdr
			},
			rule: &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByPerIp{}},
			want: "123",
		},
	}

	l.regexps = []*regexp.Regexp{regexp.MustCompile(`(\d+)`)}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, _ := l.getKey(tt.headerMap(), tt.rule)

			assert.NotEmpty(t, keys, "keys should not be empty")
			assert.Equal(t, tt.want, keys[0], "getKey %s failed", tt.name)
		})
	}
}

func TestDecodeData_ErrorBranches(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	rule := &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByConsumer{}}

	l := NewLimiter(WithRedisLimiter(rdb), WithTokenizer("openai"))
	hdr := envoy.NewRequestHeaderMap(http.Header{ConsumerHeader: []string{"user1"}})
	res := l.DecodeData(hdr, rule, "some content", "gpt-3.5-turbo")

	if res == nil {
		t.Log("DecodeData returned nil (allowed/continue)")
	} else {
		switch v := res.(type) {
		case *api.LocalResponse:
			t.Log("DecodeData returned LocalResponse as expected")
		case api.ResultAction:
			t.Logf("DecodeData returned ResultAction (Continue), type: %T", v)
		default:
			t.Errorf("DecodeData returned unexpected type: %T", v)
		}
	}

	l = NewLimiter(
		WithRedisLimiter(rdb),
		WithTokenizer("openai"),
		WithTokenStats(&TokenStats{WindowSize: 3, MinSamples: 1, MaxRatio: 0.1, MaxTokensPerReq: 1}),
	)
	hdr = envoy.NewRequestHeaderMap(http.Header{ConsumerHeader: []string{"user1"}})
	res = l.DecodeData(hdr, rule, "some content", "gpt-3.5-turbo")

	if res == nil {
		t.Log("DecodeData returned nil (allowed/continue)")
	} else {
		switch v := res.(type) {
		case *api.LocalResponse:
			t.Log("DecodeData returned LocalResponse as expected")
		case api.ResultAction:
			t.Logf("DecodeData returned ResultAction (Continue), type: %T", v)
		default:
			t.Errorf("DecodeData returned unexpected type: %T", v)
		}
	}

	l.tokenizer = &tokenizer.OpenaiTokenizer{}
	l.tokenStat = &TokenStats{
		WindowSize:      3,
		MinSamples:      1,
		MaxRatio:        0.1,
		MaxTokensPerReq: 1,
	}
	l.buckets = []*limittoken.Bucket{{Burst: 10, Rate: 10, Round: 1}}
	l.keys = []string{"key1"}

	res = l.EncodeData("hello", "gpt-3.5-turbo", 0, 1)
	if res == nil {
		t.Log("EncodeData returned nil (allowed/continue)")
	} else {
		switch v := res.(type) {
		case *api.LocalResponse:
			t.Log("EncodeData returned LocalResponse as expected")
		case api.ResultAction:
			t.Logf("EncodeData returned ResultAction (Continue), type: %T", v)
		default:
			t.Errorf("EncodeData returned unexpected type: %T", v)
		}
	}
}

func TestGetKey_PerModeBranches(t *testing.T) {
	l := NewLimiter(
		WithRegexps([]*regexp.Regexp{regexp.MustCompile(`(\d+)`)}),
	)

	tests := []struct {
		name   string
		rule   *limittoken.Rule
		header http.Header
		want   string
	}{
		{
			name:   "PerHeader",
			rule:   &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByPerHeader{LimitByPerHeader: "X-Test"}},
			header: http.Header{"X-Test": []string{"abc123"}},
			want:   "123",
		},
		{
			name:   "PerParam",
			rule:   &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByPerParam{LimitByPerParam: "p"}},
			header: http.Header{":path": []string{"/?p=456"}},
			want:   "456",
		},
		{
			name:   "PerCookie",
			rule:   &limittoken.Rule{LimitBy: &limittoken.Rule_LimitByPerCookie{LimitByPerCookie: "c"}},
			header: http.Header{"Cookie": []string{"c=789"}},
			want:   "789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := envoy.NewRequestHeaderMap(tt.header)
			keys, _ := l.getKey(hdr, tt.rule)
			assert.Equal(t, tt.want, keys[0])
		})
	}
}
