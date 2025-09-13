package limiter

import (
	"context"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"net/http"
	"regexp"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/plugins/plugins/limitToken/tokenizer"
	"mosn.io/htnn/types/plugins/limitToken"
)

func TestOptions(t *testing.T) {
	r := regexp.MustCompile(`(\d+)`)
	// 启动内存 Redis
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

	assert.Equal(t, "reject", l.rejectedMsg, "rejectedMsg 应该被正确设置")
	assert.Equal(t, 499, l.rejectedCode, "rejectedCode 应该被正确设置")
	assert.Equal(t, ts, l.tokenStat, "tokenStat 应该被正确设置")
}

func TestDecodeDataAndEncodeData(t *testing.T) {
	l := NewLimiter()
	l.tokenizer = &tokenizer.OpenaiTokenizer{}
	l.tokenStat = &TokenStats{
		WindowSize: 5, MinSamples: 1,
		MaxRatio: 2, MaxTokensPerReq: 100, ExceedFactor: 2,
	}
	l.buckets = []*limitToken.Bucket{{Burst: 10, Rate: 100, Round: 1}}

	httpHdr := http.Header{"x-mse-consumer": []string{"user1"}}
	hdr := envoy.NewRequestHeaderMap(httpHdr)
	hdr.Set(":authority", "127.0.0.1")

	rule := &limitToken.Rule{LimitBy: &limitToken.Rule_LimitByConsumer{}}

	// 模拟真实 OpenAI 请求 JSON
	messages := []tokenizer.OpenaiPromptMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello! How are you?"},
	}
	messagesStr, err := json.Marshal(messages)
	assert.NoError(t, err, "marshal messages should not fail")

	res := l.DecodeData(hdr, rule, string(messagesStr), "gpt-3.5-turbo-0613")
	assert.Equal(t, api.Continue, res, "DecodeData 应该返回 Continue")

	res = l.EncodeData("你好，这里是Go函数示例", "gpt-3.5-turbo-0613", 20, 10)
	assert.Equal(t, api.Continue, res, "EncodeData 应该返回 Continue")
}

func TestEncodeStreamData(t *testing.T) {
	// 启动内存 Redis
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
	l.buckets = []*limitToken.Bucket{{Burst: 100, Rate: 10, Round: 1}}
	l.predictCompletionToken = 5

	messages := []tokenizer.OpenaiPromptMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "system", Content: "Hello! How are you?"},
	}
	messagesStr, err := json.Marshal(messages)
	assert.NoError(t, err, "marshal messages should not fail")

	// 非最终 chunk
	res := l.EncodeStreamData(string(messagesStr), "gpt-3.5-turbo-0613", false)
	assert.Equal(t, api.Continue, res, "EncodeStreamData(non-final) 应该返回 Continue")

	// 最终 chunk
	res = l.EncodeStreamData(string(messagesStr), "gpt-3.5-turbo-0613", true)
	assert.Equal(t, api.Continue, res, "EncodeStreamData(final) 应该返回 Continue")
}

func TestTokenStats(t *testing.T) {
	s := &TokenStats{WindowSize: 3, MinSamples: 2, MaxRatio: 2, MaxTokensPerReq: 50, ExceedFactor: 2}

	// 样本不足 => 走 MaxRatio
	s.Add(10, 20)
	if !s.IsExceeded(10) {
		t.Errorf("expected true when sample insufficient")
	}

	// 添加足够样本
	s.Add(10, 30)
	s.Add(20, 40)
	s.Add(30, 60) // 覆盖旧值

	if v := s.PredictCompletionTokens(10); v <= 0 {
		t.Errorf("predict should >0, got %d", v)
	}
}

func TestGetKey(t *testing.T) {
	l := NewLimiter()
	tests := []struct {
		name      string
		headerMap func() api.RequestHeaderMap
		rule      *limitToken.Rule
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
			rule: &limitToken.Rule{LimitBy: &limitToken.Rule_LimitByHeader{LimitByHeader: "X-header"}},
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
			rule: &limitToken.Rule{LimitBy: &limitToken.Rule_LimitByParam{LimitByParam: "p"}},
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
			rule: &limitToken.Rule{LimitBy: &limitToken.Rule_LimitByCookie{LimitByCookie: "c"}},
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
			rule: &limitToken.Rule{LimitBy: &limitToken.Rule_LimitByPerIp{}},
			want: "123",
		},
	}

	l.regexps = []*regexp.Regexp{regexp.MustCompile(`(\d+)`)}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, _ := l.getKey(tt.headerMap(), tt.rule)

			assert.NotEmpty(t, keys, "keys 不应该为空")
			assert.Equal(t, tt.want, keys[0], "getKey %s fail", tt.name)
		})
	}
}
