package limiter

import (
	"context"
	"fmt"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/plugins/limitToken/tokenizer"
	"mosn.io/htnn/types/plugins/limitToken"
	"regexp"
	"sort"
	"time"
)

const (
	ConsumerHeader string = "x-mse-consumer"
)

// Limiter holds the configuration and runtime state for token rate limiting
type Limiter struct {
	rejectedMsg            string
	rejectedCode           int
	keys                   []string
	buckets                []*limitToken.Bucket
	promptToken            int
	predictCompletionToken int
	totalCompletionToken   int
	tokenizer              tokenizer.Tokenizer
	tokenStat              *TokenStats
	regexps                []*regexp.Regexp
	redisLimiter           *redis_rate.Limiter
}

// Option defines a functional option for Limiter
type Option func(*Limiter)

// WithRegexps sets regex patterns for key extraction
func WithRegexps(r []*regexp.Regexp) Option {
	return func(l *Limiter) {
		l.regexps = r
	}
}

// WithRedisLimiter sets a Redis-based rate limiter
func WithRedisLimiter(rdb *redis.Client) Option {
	return func(l *Limiter) {
		l.redisLimiter = redis_rate.NewLimiter(rdb)
	}
}

// WithTokenizer sets the tokenizer provider
func WithTokenizer(provider string) Option {
	return func(l *Limiter) {
		switch provider {
		case "openai":
			l.tokenizer = &tokenizer.OpenaiTokenizer{}
		default:
			l.tokenizer = &tokenizer.OpenaiTokenizer{}
		}
	}
}

// WithTokenStats sets the token statistics for prediction
func WithTokenStats(tokenStat *TokenStats) Option {
	return func(l *Limiter) {
		l.tokenStat = tokenStat
	}
}

// WithRejectedMsg sets the message returned when a request is rejected
func WithRejectedMsg(rejectedMsg string) Option {
	return func(l *Limiter) {
		l.rejectedMsg = rejectedMsg
	}
}

// WithRejectedCode sets the HTTP status code returned when a request is rejected
func WithRejectedCode(rejectedCode int) Option {
	return func(l *Limiter) {
		l.rejectedCode = rejectedCode
	}
}

// NewLimiter creates a new Limiter instance with provided options
func NewLimiter(opts ...Option) *Limiter {
	l := &Limiter{
		rejectedMsg:  "You are rate limited", // default
		rejectedCode: 429,                    // default
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// DecodeData processes a request and applies rate limiting
func (l *Limiter) DecodeData(headers api.RequestHeaderMap, rule *limitToken.Rule, content, model string) api.ResultAction {
	l.buckets = rule.Buckets
	// Get the keys for rate limiting
	keys, err := l.getKey(headers, rule)
	if err != nil {
		api.LogErrorf("error getting key: %v", err)
		return api.Continue
	}

	promptToken, err := l.tokenizer.GetToken(content, model)
	if err != nil {
		return nil
	}

	// Check if prompt tokens exceed default limit
	if exceeded := l.tokenStat.IsExceeded(promptToken); !exceeded {
		api.LogErrorf("token exceeded for prompt: %d", promptToken)
		return &api.LocalResponse{
			Code: l.rejectedCode,
			Msg:  l.rejectedMsg,
		}
	}

	// Predict completion tokens
	predictCompletionToken := l.tokenStat.PredictCompletionTokens(promptToken)
	// Check if token usage is within allowed rate
	if ok := l.tokenRate(keys, predictCompletionToken); !ok {
		api.LogErrorf("token rate exceeded in DecodeRequest, keys: %v, token: %d", keys, predictCompletionToken)
		return &api.LocalResponse{
			Code: l.rejectedCode,
			Msg:  l.rejectedMsg,
		}
	}
	l.promptToken = promptToken
	l.predictCompletionToken = predictCompletionToken
	l.keys = keys

	return api.Continue
}

// EncodeData applies rate limiting for response data
func (l *Limiter) EncodeData(content, model string, completionToken, promptToken int) api.ResultAction {
	var err error
	if completionToken == 0 {
		completionToken, err = l.tokenizer.GetToken(content, model)
		if err != nil {
			return nil
		}
	}

	// Calculate actual difference and check rate
	tokenGap := completionToken - l.predictCompletionToken
	if ok := l.tokenRate(l.keys, tokenGap); !ok {
		api.LogErrorf("token rate exceeded in EncodeData")
		return &api.LocalResponse{
			Code: l.rejectedCode,
			Msg:  l.rejectedMsg,
		}
	}

	// Report statistics
	l.tokenStat.Add(promptToken, completionToken)

	return api.Continue
}

// EncodeStreamData applies rate limiting for streaming responses
func (l *Limiter) EncodeStreamData(content, model string, isEnd bool) api.ResultAction {
	completionToken, err := l.tokenizer.GetToken(content, model)
	if err != nil {
		return nil
	}

	l.totalCompletionToken += completionToken

	if isEnd {
		// Check if total tokens exceed limit
		tokenGap := l.totalCompletionToken - l.predictCompletionToken
		// Report statistics and reset counters
		defer func() {
			l.tokenStat.Add(l.promptToken, l.totalCompletionToken)
			l.totalCompletionToken = 0
			l.promptToken = 0
			l.predictCompletionToken = 0
		}()

		if ok := l.tokenRate(l.keys, tokenGap); !ok {
			return &api.LocalResponse{
				Code: l.rejectedCode,
				Msg:  l.rejectedMsg,
			}
		}
	}

	return api.Continue
}

// tokenRate checks whether the token rate is within allowed limits
func (l *Limiter) tokenRate(keys []string, promptTokenLength int) bool {
	ctx := context.Background()

	for _, key := range keys {
		for i := range l.buckets {
			b := l.buckets[i]

			limit := redis_rate.Limit{
				Rate:   int(b.Rate),
				Burst:  int(b.Burst),
				Period: time.Duration(b.Round) * time.Second,
			}

			// Create separate Redis key for each period
			redisKey := fmt.Sprintf("%s:%ds", key, b.Round)

			res, err := l.redisLimiter.AllowN(ctx, redisKey, limit, promptTokenLength)
			if err != nil {
				api.LogErrorf("limitReq filter Redis error: %v", err)
				return false
			}

			if res.Allowed == 0 {
				api.LogInfof("limitReq filter, key: %s, bucket: %ds, denied: too many requests", key, b.Round)
				return false
			}

			api.LogInfof("limitReq filter, key: %s, bucket: %ds, allowed: %d, retry after: %s",
				key, b.Round, res.Allowed, res.RetryAfter)
		}
	}

	return true
}

// getKey extracts the key(s) from headers according to the rule
func (l *Limiter) getKey(headers api.RequestHeaderMap, rule *limitToken.Rule) ([]string, error) {
	var raw string
	var ok, isMatchMode bool

	switch v := rule.LimitBy.(type) {
	case *limitToken.Rule_LimitByHeader:
		raw, ok = headers.Get(v.LimitByHeader)

	case *limitToken.Rule_LimitByParam:
		raw, ok = l.getQueryValue(headers, v.LimitByParam)

	case *limitToken.Rule_LimitByCookie:
		raw, ok = l.getCookieValue(headers, v.LimitByCookie)

	case *limitToken.Rule_LimitByConsumer:
		raw, ok = headers.Get(ConsumerHeader)

	case *limitToken.Rule_LimitByPerIp:
		raw, ok = headers.Host(), true
		isMatchMode = true

	case *limitToken.Rule_LimitByPerHeader:
		raw, ok = headers.Get(v.LimitByPerHeader)
		isMatchMode = true

	case *limitToken.Rule_LimitByPerParam:
		raw, ok = l.getQueryValue(headers, v.LimitByPerParam)
		isMatchMode = true

	case *limitToken.Rule_LimitByPerCookie:
		raw, ok = l.getCookieValue(headers, v.LimitByPerCookie)
		isMatchMode = true

	case *limitToken.Rule_LimitByPerConsumer:
		raw, ok = headers.Get(ConsumerHeader)
		isMatchMode = true

	default:
		return nil, fmt.Errorf("unknown limit type: %v", rule.LimitBy)
	}

	if !ok {
		return nil, nil
	}

	if isMatchMode {
		result := make([]string, 0, len(l.regexps))
		for _, reg := range l.regexps {
			if matches := reg.FindStringSubmatch(raw); len(matches) > 1 {
				result = append(result, matches[1])
			}
		}
		return result, nil
	}

	return []string{raw}, nil
}

// getQueryValue retrieves the value of a query parameter from the request URL
func (l *Limiter) getQueryValue(headers api.RequestHeaderMap, key string) (string, bool) {
	val := headers.URL().Query().Get(key)
	if val != "" {
		return val, true
	}
	return "", false
}

// getCookieValue retrieves the value of a cookie from the request headers
func (l *Limiter) getCookieValue(headers api.RequestHeaderMap, key string) (string, bool) {
	cookie := headers.Cookie(key)
	if cookie != nil {
		return cookie.String(), true
	}
	return "", false
}

// TokenPair stores the prompt and completion token counts for a request
type TokenPair struct {
	Prompt     int
	Completion int
}

// TokenStats tracks statistics for prompt and completion tokens to enforce limits and make predictions
type TokenStats struct {
	WindowSize      int         // maximum number of samples in the sliding window
	Data            []TokenPair // token usage data
	index           int         // current write index in the circular buffer
	full            bool        // indicates if the sliding window is full
	MinSamples      int         // minimum number of samples required to perform predictions
	MaxRatio        float64     // maximum default ratio of completion tokens to prompt tokens
	MaxTokensPerReq int         // maximum allowed tokens per request
	ExceedFactor    float64     // factor to allow exceeding predicted token usage
}

// Add records a new token usage pair in the sliding window
func (s *TokenStats) Add(promptTokens, completionTokens int) {
	// Ensure promptTokens is at least 1 to avoid panic
	if promptTokens <= 0 {
		promptTokens = 1
	}

	if len(s.Data) < s.WindowSize {
		s.Data = append(s.Data, TokenPair{Prompt: promptTokens, Completion: completionTokens})
	} else {
		// Overwrite the oldest entry in the circular buffer
		s.Data[s.index] = TokenPair{Prompt: promptTokens, Completion: completionTokens}
		s.index = (s.index + 1) % s.WindowSize
		s.full = true
	}
}

// IsExceeded checks whether the given prompt token count exceeds allowed limits
func (s *TokenStats) IsExceeded(promptTokens int) bool {
	if len(s.Data) < s.MinSamples {
		// If insufficient samples, use default maximum ratio
		return float64(promptTokens)*s.MaxRatio < float64(s.MaxTokensPerReq)
	}

	var ratios []float64
	var completions []int
	for _, pair := range s.Data {
		// Avoid division by zero
		ratios = append(ratios, float64(pair.Completion)/float64(max(1, pair.Prompt)))
		completions = append(completions, pair.Completion)
	}
	sort.Float64s(ratios)
	sort.Ints(completions)

	// 95th percentile ratio for expected completion token estimation
	posRatio := int(0.95 * float64(len(ratios)))
	expectedCompletion := float64(promptTokens) * ratios[posRatio]

	// 95th percentile of actual completion tokens
	posCompletion := int(0.95 * float64(len(completions)))
	completionP95 := float64(completions[posCompletion])

	// Limit if expected exceeds the historical P95 multiplied by exceed factor
	return expectedCompletion < completionP95*s.ExceedFactor
}

// PredictCompletionTokens estimates the number of completion tokens for a given prompt
func (s *TokenStats) PredictCompletionTokens(promptTokens int) int {
	if len(s.Data) < s.MinSamples {
		estimated := float64(promptTokens) * s.MaxRatio
		if estimated > float64(s.MaxTokensPerReq) {
			estimated = float64(s.MaxTokensPerReq)
		}
		return int(estimated)
	}

	var sumRatio float64
	var count int
	for _, pair := range s.Data {
		if pair.Prompt > 0 {
			sumRatio += float64(pair.Completion) / float64(pair.Prompt)
			count++
		}
	}
	if count == 0 {
		return int(float64(promptTokens) * s.MaxRatio)
	}

	avgRatio := sumRatio / float64(count)
	estimated := float64(promptTokens) * avgRatio
	if estimated > float64(s.MaxTokensPerReq) {
		estimated = float64(s.MaxTokensPerReq)
	}
	return int(estimated)
}
