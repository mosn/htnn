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

package hmac_auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"hash"
	"slices"
	"sort"
	"strings"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/request"
)

func configFactory(c interface{}) api.FilterFactory {
	conf := c.(*Config)
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
	config    *Config
	consumer  *ConsumerConfig
}

// This plugin uses the same hmac auth scheme as APISIX:
// https://apisix.apache.org/docs/apisix/plugins/hmac-auth/#generating-the-signature
const (
	DateHeader      = "date"
	SignatureHeader = "x-hmac-signature"
	AccessKeyHeader = "x-hmac-access-key"
	// TODO: support algorithm / signed header filters
)

func (f *filter) getSignContent(header api.RequestHeaderMap, accessKey string) string {
	dh := DateHeader
	if f.config.DateHeader != "" {
		dh = f.config.DateHeader
	}
	date, _ := header.Get(dh)
	url := request.GetUrl(header)
	path := url.Path
	if path == "" {
		path = "/"
	}
	query := url.Query()
	sortedQuery := make([][2]string, 0, len(query))
	for k, v := range query {
		if len(v) == 1 {
			sortedQuery = append(sortedQuery, [2]string{k, v[0]})
		} else {
			for _, vv := range v {
				sortedQuery = append(sortedQuery, [2]string{k, vv})
			}
		}
	}
	sort.Slice(sortedQuery, func(i, j int) bool {
		if sortedQuery[i][0] == sortedQuery[j][0] {
			return sortedQuery[i][1] < sortedQuery[j][1]
		}
		return sortedQuery[i][0] < sortedQuery[j][0]
	})

	buf := strings.Builder{}
	buf.WriteString(header.Method())
	buf.WriteByte('\n')
	buf.WriteString(path)
	buf.WriteByte('\n')
	for i, kv := range sortedQuery {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(kv[0])
		buf.WriteByte('=')
		buf.WriteString(kv[1])
	}
	buf.WriteByte('\n')
	buf.WriteString(accessKey)
	buf.WriteByte('\n')
	buf.WriteString(date)
	buf.WriteByte('\n')
	for _, h := range f.consumer.SignedHeaders {
		hs := header.Values(h)
		slices.Sort(hs)
		for _, v := range hs {
			buf.WriteString(h)
			buf.WriteByte(':')
			buf.WriteString(v)
			buf.WriteByte('\n')
		}
	}

	return buf.String()
}

func (f *filter) sign(value []byte) string {
	secret := []byte(f.consumer.SecretKey)

	var hash hash.Hash
	switch f.consumer.Algorithm {
	case Algorithm_HMAC_SHA256:
		hash = hmac.New(sha256.New, secret)
	case Algorithm_HMAC_SHA384:
		hash = hmac.New(sha512.New384, secret)
	case Algorithm_HMAC_SHA512:
		hash = hmac.New(sha512.New, secret)
	}
	hash.Write(value)
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	config := f.config
	akh := AccessKeyHeader
	if config.AccessKeyHeader != "" {
		akh = f.config.AccessKeyHeader
	}
	sh := SignatureHeader
	if config.SignatureHeader != "" {
		sh = f.config.SignatureHeader
	}

	// We only cares about one of the headers if multiple is given.
	// The others will be dropped.
	accessKey, ok := headers.Get(akh)
	if !ok {
		return api.Continue
	}

	c, ok := f.callbacks.LookupConsumer(Name, accessKey)
	if !ok {
		api.LogInfof("can not find consumer with access key %s in %s", accessKey, akh)
		return &api.LocalResponse{Code: 401, Msg: "invalid access key"}
	}

	f.consumer = c.PluginConfig(Name).(*ConsumerConfig)
	signature, _ := headers.Get(sh)
	signContent := f.getSignContent(headers, accessKey)
	generatedSign := f.sign([]byte(signContent))
	if signature != generatedSign {
		api.LogInfof("signature mismatch: expected %s != actual %s, source: %q",
			signature, generatedSign, signContent)
		return &api.LocalResponse{Code: 401, Msg: "invalid signature"}
	}

	// drop sensitive headers
	headers.Del(akh)
	headers.Del(sh)
	f.callbacks.SetConsumer(c)
	return api.Continue
}
