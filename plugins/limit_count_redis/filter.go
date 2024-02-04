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

package limit_count_redis

import (
	"context"
	"fmt"
	"net/http"

	"mosn.io/htnn/pkg/expr"
	"mosn.io/htnn/pkg/ext"
	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/request"
)

func configFactory(c interface{}) api.FilterFactory {
	conf := c.(*config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
}

func (f *filter) getKey(script expr.Script, headers api.RequestHeaderMap) string {
	var key string
	if script != nil {
		res, err := script.EvalWithRequest(f.callbacks, headers)
		if err == nil {
			key = res.(string)
		}
	}
	if key == "" {
		api.LogInfo("limitCountRedis filter uses client IP as key because the configured key is empty")
		key = request.GetRemoteIP(f.callbacks.StreamInfo())
	}
	return key
}

var (
	redisScript = ext.CutSpace(`
	local res={}
	for i=1,%d do
		local ttl=redis.call('ttl',KEYS[i])
		if ttl<0 then
			redis.call('set',KEYS[i],ARGV[i*2-1]-1,'EX',ARGV[i*2])
			res[i*2-1]=ARGV[i*2-1]-1
			res[i*2]=ARGV[i*2]
		else
			res[i*2-1]=redis.call('incrby',KEYS[i],-1)
			res[i*2]=ttl
		end
	end
	return res
	`)
)

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	ctx := context.Background()
	config := f.config
	n := len(config.limiters)
	keys := make([]string, n)
	args := make([]interface{}, n*2)
	for i, limiter := range config.limiters {
		key := f.getKey(limiter.script, headers)
		keys[i] = limiter.prefix + "|" + key

		api.LogInfof("limitCountRedis filter, key: %s", key)

		args[i*2] = limiter.count
		args[i*2+1] = limiter.timeWindow
	}

	cmd := config.client.Eval(ctx, fmt.Sprintf(redisScript, n), keys, args...)
	res, err := cmd.Result()
	if err != nil {
		api.LogErrorf("failed to limit count: %v", err)

		if config.FailureModeDeny {
			return &api.LocalResponse{Code: 503}
		}
		return api.Continue
	}

	ress := res.([]interface{})
	for i := 0; i < len(config.limiters); i++ {
		remain := ress[2*i].(int64)
		if remain < 0 {
			// TODO: add X-RateLimit headers
			hdr := http.Header{}
			// TODO: add option to disable x-envoy-ratelimited
			hdr.Set("X-Envoy-Ratelimited", "true")
			return &api.LocalResponse{Code: 429, Header: hdr}
		}
	}

	return api.Continue
}
