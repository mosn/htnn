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

package limitcountredis

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/redis/go-redis/v9"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/pkg/stringx"
	"mosn.io/htnn/types/pkg/expr"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config

	ress []interface{}
}

func (f *filter) getKey(script expr.Script, headers api.RequestHeaderMap) string {
	var key string
	if script != nil {
		res, err := script.EvalWithRequest(f.callbacks, headers)
		if err == nil {
			key = res.(string)
		}
		if key == "" {
			api.LogInfo("limitCountRedis filter uses client IP as key because the configured key is empty")
		}
	}
	if key == "" {
		key = f.callbacks.StreamInfo().DownstreamRemoteParsedAddress().IP
	}
	return key
}

var (
	redisScript = stringx.CutSpace(`
	local res={}
	for i=1,%d do
		local ttl=redis.call('ttl',KEYS[i])
		if ttl<0 then
			redis.call('set',KEYS[i],ARGV[i*2-1]-1,'EX',ARGV[i*2])
			res[i*2-1]=ARGV[i*2-1]-1
			res[i*2]=tonumber(ARGV[i*2])
		else
			res[i*2-1]=redis.call('incrby',KEYS[i],-1)
			res[i*2]=ttl
		end
	end
	return res
	`)

	redisSingleScript = stringx.CutSpace(`
	local res={}
	local ttl=redis.call('ttl',KEYS[1])
	if ttl<0 then
		redis.call('set',KEYS[1],ARGV[1]-1,'EX',ARGV[2])
		res[1]=ARGV[1]-1
		res[2]=tonumber(ARGV[2])
	else
		res[1]=redis.call('incrby',KEYS[1],-1)
		res[2]=ttl
	end
	return res
	`)
)

func (f *filter) limitCountErr(err error) api.ResultAction {
	config := f.config
	api.LogErrorf("failed to limit count: %v", err)

	if config.FailureModeDeny {
		status := 500 // follow the behavior of Envoy
		if config.StatusOnError != 0 {
			status = int(config.StatusOnError)
		}
		return &api.LocalResponse{Code: status}
	}
	return api.Continue
}

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

	var ress []interface{}

	if config.GetCluster() != nil {
		// Redis cluster doesn't support operation across multiple slots, so we have to
		// use pipeline to send the request one by one. We can't use hash tag because
		// this will cause the key imbalence.
		cmds, err := config.clusterClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for i, k := range keys {
				pipe.Eval(ctx, redisSingleScript, []string{k}, args[i*2], args[i*2+1])
			}
			return nil
		})
		if err != nil {
			return f.limitCountErr(err)
		}

		ress = make([]interface{}, 2*len(cmds))
		for i, cmd := range cmds {
			res := cmd.(*redis.Cmd).Val().([]interface{})
			ress[i*2] = res[0]
			ress[i*2+1] = res[1]
		}

	} else {
		cmd := config.client.Eval(ctx, fmt.Sprintf(redisScript, n), keys, args...)
		res, err := cmd.Result()
		if err != nil {
			return f.limitCountErr(err)
		}

		ress = res.([]interface{})
	}
	f.ress = ress

	for i := range config.limiters {
		remain := ress[2*i].(int64)
		if remain < 0 {
			hdr := http.Header{}
			if config.EnableLimitQuotaHeaders {
				hdr.Set("x-envoy-ratelimited", "true")
			}
			status := 429
			if config.RateLimitedStatus >= 400 { // follow the behavior of Envoy
				status = int(config.RateLimitedStatus)
			}
			return &api.LocalResponse{Code: status, Header: hdr}
		}
	}

	return api.Continue
}

func (f *filter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	config := f.config
	if !config.EnableLimitQuotaHeaders {
		return api.Continue
	}
	if len(f.ress) == 0 {
		// If the redis call is failed, we don't know the correct value of limit quota headers.
		return api.Continue
	}

	var minCount uint32
	var minRemain int64 = math.MaxUint32
	var minTTL int64
	for i, lim := range f.config.limiters {
		remain := f.ress[2*i].(int64)
		ttl := f.ress[2*i+1].(int64)

		if remain < minRemain {
			minRemain = remain
			minCount = lim.count
			minTTL = ttl
		}
	}

	// According to the RFC, these headers MUST NOT occur multiple times.
	headers.Set("x-ratelimit-limit", fmt.Sprintf("%d, %s", minCount, config.quotaPolicy))
	if minRemain <= 0 {
		headers.Set("x-ratelimit-remaining", "0")
	} else {
		headers.Set("x-ratelimit-remaining", strconv.FormatInt(minRemain, 10))
	}
	headers.Set("x-ratelimit-reset", strconv.FormatInt(minTTL, 10))
	return api.Continue
}
