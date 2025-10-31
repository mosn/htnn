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

package limittoken

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/limittoken"
)

func TestConfig_Init(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	tests := []struct {
		name      string
		conf      *config
		wantErr   bool
		checkFunc func(t *testing.T, conf *config)
	}{
		{
			name: "Default TokenStats applied",
			conf: &config{
				CustomConfig: limittoken.CustomConfig{
					Config: limittoken.Config{
						Redis: &limittoken.RedisConfig{ServiceAddr: s.Addr()},
						Rule:  &limittoken.Rule{},
						ExtractorConfig: &limittoken.Config_GjsonConfig{
							GjsonConfig: &limittoken.GjsonConfig{RequestContentPath: "messages"},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, conf *config) {
				assert.Equal(t, DefaultWindowSize, conf.tokenStats.WindowSize)
				assert.Equal(t, DefaultMinSamples, conf.tokenStats.MinSamples)
				assert.Equal(t, DefaultMaxTokensPerReq, conf.tokenStats.MaxTokensPerReq)
			},
		},
		{
			name: "Custom TokenStats applied",
			conf: &config{
				CustomConfig: limittoken.CustomConfig{
					Config: limittoken.Config{
						Redis: &limittoken.RedisConfig{ServiceAddr: s.Addr()},
						Rule:  &limittoken.Rule{},
						TokenStats: &limittoken.TokenStatsConfig{
							WindowSize:      10,
							MinSamples:      5,
							MaxRatio:        2.5,
							MaxTokensPerReq: 500,
							ExceedFactor:    3.0,
						},
						ExtractorConfig: &limittoken.Config_GjsonConfig{
							GjsonConfig: &limittoken.GjsonConfig{RequestContentPath: "messages"},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, conf *config) {
				assert.Equal(t, 10, conf.tokenStats.WindowSize)
				assert.Equal(t, 5, conf.tokenStats.MinSamples)
				assert.Equal(t, 500, conf.tokenStats.MaxTokensPerReq)
				assert.Equal(t, 3.0, conf.tokenStats.ExceedFactor)
			},
		},
		{
			name: "Valid regexps",
			conf: &config{
				CustomConfig: limittoken.CustomConfig{
					Config: limittoken.Config{
						Redis: &limittoken.RedisConfig{ServiceAddr: s.Addr()},
						Rule:  &limittoken.Rule{Keys: []string{`^user-\d+$`}},
						ExtractorConfig: &limittoken.Config_GjsonConfig{
							GjsonConfig: &limittoken.GjsonConfig{RequestContentPath: "messages"},
						},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, conf *config) {
				assert.Len(t, conf.regexps, 1)
				assert.True(t, conf.regexps[0].MatchString("user-123"))
			},
		},
		{
			name: "Invalid regexps should fail",
			conf: &config{
				CustomConfig: limittoken.CustomConfig{
					Config: limittoken.Config{
						Redis: &limittoken.RedisConfig{ServiceAddr: s.Addr()},
						Rule:  &limittoken.Rule{Keys: []string{"[invalid"}},
						ExtractorConfig: &limittoken.Config_GjsonConfig{
							GjsonConfig: &limittoken.GjsonConfig{RequestContentPath: "messages"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid redis addr should fail",
			conf: &config{
				CustomConfig: limittoken.CustomConfig{
					Config: limittoken.Config{
						Redis: &limittoken.RedisConfig{ServiceAddr: "127.0.0.1:6390"}, // 错误端口
						Rule:  &limittoken.Rule{},
						ExtractorConfig: &limittoken.Config_GjsonConfig{
							GjsonConfig: &limittoken.GjsonConfig{RequestContentPath: "messages"},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Init(envoy.NewFilterCallbackHandler())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, tt.conf)
				}
			}
		})
	}
}

func TestInitRedisLimiter_Ping(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	conf := &config{
		CustomConfig: limittoken.CustomConfig{
			Config: limittoken.Config{
				Redis: &limittoken.RedisConfig{ServiceAddr: s.Addr()},
				Rule:  &limittoken.Rule{},
			},
		},
	}
	err = conf.initRedisLimiter()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = conf.rdb.Set(ctx, "k", "v", 0).Err()
	assert.NoError(t, err)
	val, err := conf.rdb.Get(ctx, "k").Result()
	assert.NoError(t, err)
	assert.Equal(t, "v", val)
}

func TestInitExtractor_OK(t *testing.T) {
	conf := &config{
		CustomConfig: limittoken.CustomConfig{
			Config: limittoken.Config{
				Redis: &limittoken.RedisConfig{},
				Rule:  &limittoken.Rule{},
				ExtractorConfig: &limittoken.Config_GjsonConfig{
					GjsonConfig: &limittoken.GjsonConfig{RequestContentPath: "messages"},
				},
			},
		},
	}
	err := conf.initExtractor()
	assert.NoError(t, err)
	assert.NotNil(t, conf.extractor)
}

func TestPlugin_Type(t *testing.T) {
	p := &plugin{}
	assert.Equal(t, plugins.TypeTraffic, p.Type())
}
