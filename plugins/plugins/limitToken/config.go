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
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/plugins/plugins/limitToken/extractor"
	"mosn.io/htnn/plugins/plugins/limitToken/limiter"
	"mosn.io/htnn/types/plugins/limitToken"
	"reflect"
	"regexp"
)

const (
	DefaultWindowSize      = 1000
	DefaultMinSamples      = 10
	DefaultMaxRatio        = 4.0
	DefaultMaxTokensPerReq = 2000
	DefaultExceedFactor    = 1.5
)

func init() {
	plugins.RegisterPlugin(limitToken.Name, &plugin{})
}

// plugin implements the limitToken.Plugin interface
type plugin struct {
	limitToken.Plugin
}

// Factory returns the filter factory
func (p *plugin) Factory() api.FilterFactory {
	return factory
}

// Config returns the plugin configuration
func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

// config holds the runtime configuration of the limitToken plugin
type config struct {
	limitToken.CustomConfig
	rdb        *redis.Client
	tokenStats *limiter.TokenStats
	extractor  extractor.Extractor
	regexps    []*regexp.Regexp
}

// Init initializes the plugin configuration
func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	if err := conf.initTokenStats(); err != nil {
		return err
	}

	if err := conf.initRedisLimiter(); err != nil {
		return err
	}

	if err := conf.initRegexps(); err != nil {
		return err
	}

	if err := conf.initExtrator(); err != nil {
		return err
	}

	return nil
}

// initRedisLimiter initializes the Redis client used for distributed rate limiting
func (conf *config) initRedisLimiter() error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.ServiceAddr,
		Username: conf.Redis.Username,
		Password: conf.Redis.Password,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	conf.rdb = rdb

	return nil
}

// initExtrator creates the Extractor (data extractor) based on ExtractorConfig
func (conf *config) initExtrator() error {
	extractorTypeName := reflect.TypeOf(conf.ExtractorConfig).String()
	newExtractor, err := extractor.NewExtractor(extractorTypeName, conf.ExtractorConfig)
	if err != nil {
		api.LogErrorf("failed to create newExtractor for provider type '%s': %v", extractorTypeName, err)
		return err
	}
	conf.extractor = newExtractor

	return nil
}

// initRegexps compiles all key regex patterns used for rate limiting
func (conf *config) initRegexps() error {
	conf.regexps = make([]*regexp.Regexp, 0, len(conf.Rule.Keys))
	for _, pattern := range conf.Rule.Keys {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regexp key %q: %w", pattern, err)
		}
		conf.regexps = append(conf.regexps, re)
	}
	return nil
}

// initTokenStats initializes the token statistics used for predictive rate limiting
func (conf *config) initTokenStats() error {
	cfg := conf.TokenStats
	if cfg == nil {
		cfg = &limitToken.TokenStatsConfig{}
	}

	windowSize := int(cfg.WindowSize)
	if windowSize == 0 {
		windowSize = DefaultWindowSize
	}

	minSamples := int(cfg.MinSamples)
	if minSamples == 0 {
		minSamples = DefaultMinSamples
	}

	maxRatio := float64(cfg.MaxRatio)
	if maxRatio == 0 {
		maxRatio = DefaultMaxRatio
	}

	maxTokensPerReq := int(cfg.MaxTokensPerReq)
	if maxTokensPerReq == 0 {
		maxTokensPerReq = DefaultMaxTokensPerReq
	}

	exceedFactor := float64(cfg.ExceedFactor)
	if exceedFactor == 0 {
		exceedFactor = DefaultExceedFactor
	}

	conf.tokenStats = &limiter.TokenStats{
		WindowSize:      windowSize,
		Data:            make([]limiter.TokenPair, 0, windowSize),
		MinSamples:      minSamples,
		MaxRatio:        maxRatio,
		MaxTokensPerReq: maxTokensPerReq,
		ExceedFactor:    exceedFactor,
	}

	return nil
}
