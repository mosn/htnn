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

package limitreq

import (
	"runtime"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/time/rate"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/pkg/expr"
	"mosn.io/htnn/types/plugins/limitreq"
)

func init() {
	plugins.RegisterPlugin(limitreq.Name, &plugin{})
}

type plugin struct {
	limitreq.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	limitreq.CustomConfig

	buckets *ttlcache.Cache[string, *rate.Limiter]
	// Like traefik, we also require a max delay to avoid holding the requests for unlimited time.
	// The delay is 1/(2*rps) by default, and 500ms if the rps is less than 1.
	maxDelay time.Duration

	script expr.Script
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	period := time.Second
	if conf.Period != nil {
		period = conf.Period.AsDuration()
	}
	burst := conf.Burst
	if burst == 0 {
		burst = 1
	}

	rps := float64(time.Duration(conf.Average)*time.Second) / float64(period)
	limitRate := rate.Limit(rps)

	ttl := 1 * time.Second
	if rps < 1 {
		ttl += time.Duration(1/rps) * time.Second // ensure the bucket is not expired too early
		conf.maxDelay = 500 * time.Millisecond
	} else {
		ttl += 1 * time.Second
		conf.maxDelay = time.Second / (time.Duration(rps) * 2)
	}
	loader := ttlcache.LoaderFunc[string, *rate.Limiter](
		func(c *ttlcache.Cache[string, *rate.Limiter], key string) *ttlcache.Item[string, *rate.Limiter] {
			bucket := rate.NewLimiter(limitRate, int(burst))
			item := c.Set(key, bucket, ttlcache.DefaultTTL)
			return item
		},
	)
	buckets := ttlcache.New(
		ttlcache.WithTTL[string, *rate.Limiter](ttl),
		ttlcache.WithLoader[string, *rate.Limiter](loader),
	)
	conf.buckets = buckets
	go buckets.Start()
	runtime.SetFinalizer(conf, func(conf *config) {
		api.LogInfof("stop cache in limitReq conf: %+v", conf)
		conf.buckets.Stop()
	})

	if conf.Key != "" {
		conf.script, _ = expr.CompileCel(conf.Key, cel.StringType)
	}
	return nil
}
