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

package filtermanager

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	capi "github.com/envoyproxy/envoy/contrib/golang/common/go/api"

	"mosn.io/htnn/api/internal/consumer"
	"mosn.io/htnn/api/internal/cookie"
	"mosn.io/htnn/api/internal/pluginstate"
	"mosn.io/htnn/api/pkg/filtermanager/api"
)

type filterManagerRequestHeaderMap struct {
	capi.RequestHeaderMap

	cacheLock sync.Mutex

	u       *url.URL
	cookies map[string]*http.Cookie
}

func (headers *filterManagerRequestHeaderMap) expire(key string) {
	switch key {
	case ":path":
		headers.u = nil
	case "cookie":
		headers.cookies = nil
	}
}

func (headers *filterManagerRequestHeaderMap) Set(key, value string) {
	headers.cacheLock.Lock()
	key = strings.ToLower(key)
	headers.expire(key)
	headers.RequestHeaderMap.Set(key, value)
	headers.cacheLock.Unlock()
}

func (headers *filterManagerRequestHeaderMap) Add(key, value string) {
	headers.cacheLock.Lock()
	key = strings.ToLower(key)
	headers.expire(key)
	headers.RequestHeaderMap.Add(key, value)
	headers.cacheLock.Unlock()
}

func (headers *filterManagerRequestHeaderMap) Del(key string) {
	headers.cacheLock.Lock()
	key = strings.ToLower(key)
	headers.expire(key)
	headers.RequestHeaderMap.Del(key)
	headers.cacheLock.Unlock()
}

func (headers *filterManagerRequestHeaderMap) URL() *url.URL {
	headers.cacheLock.Lock()
	defer headers.cacheLock.Unlock()

	if headers.u == nil {
		path := headers.Path()
		u, err := url.ParseRequestURI(path)
		if err != nil {
			panic(fmt.Sprintf("unexpected bad request uri given by envoy: %v", err))
		}
		headers.u = u
	}
	return headers.u
}

// If multiple cookies match the given name, only one cookie will be returned.
func (headers *filterManagerRequestHeaderMap) Cookie(name string) *http.Cookie {
	headers.cacheLock.Lock()
	defer headers.cacheLock.Unlock()

	if headers.cookies == nil {
		cookieList := headers.Cookies()
		headers.cookies = make(map[string]*http.Cookie, len(cookieList))
		for _, c := range cookieList {
			headers.cookies[c.Name] = c
		}
	}
	return headers.cookies[name]
}

func (headers *filterManagerRequestHeaderMap) Cookies() []*http.Cookie {
	// same-name cookies may be overridden in the headers.cookies
	return cookie.ParseCookies(headers)
}

type filterManagerStreamInfo struct {
	capi.StreamInfo

	cacheLock sync.Mutex

	ipAddress *api.IPAddress
}

func (s *filterManagerStreamInfo) DownstreamRemoteParsedAddress() *api.IPAddress {
	s.cacheLock.Lock()
	if s.ipAddress == nil {
		ipport := s.StreamInfo.DownstreamRemoteAddress()
		// the IPPort given by Envoy must be valid
		ip, port, _ := net.SplitHostPort(ipport)
		p, _ := strconv.Atoi(port)
		s.ipAddress = &api.IPAddress{
			Address: ipport,
			IP:      ip,
			Port:    p,
		}
	}
	s.cacheLock.Unlock()
	return s.ipAddress
}

func (s *filterManagerStreamInfo) DownstreamRemoteAddress() string {
	return s.DownstreamRemoteParsedAddress().Address
}

type filterManagerCallbackHandler struct {
	capi.FilterCallbackHandler

	cacheLock sync.Mutex

	namespace   string
	consumer    api.Consumer
	pluginState api.PluginState

	streamInfo *filterManagerStreamInfo
}

func (cb *filterManagerCallbackHandler) Reset() {
	cb.cacheLock.Lock()

	cb.FilterCallbackHandler = nil
	// We don't reset namespace, as filterManager will only be reused in the same route,
	// which must have the same namespace.
	cb.consumer = nil
	cb.streamInfo = nil

	cb.cacheLock.Unlock()
}

func (cb *filterManagerCallbackHandler) StreamInfo() api.StreamInfo {
	cb.cacheLock.Lock()
	if cb.streamInfo == nil {
		cb.streamInfo = &filterManagerStreamInfo{
			StreamInfo: cb.FilterCallbackHandler.StreamInfo(),
		}
	}
	cb.cacheLock.Unlock()
	return cb.streamInfo
}

// Consumer getter/setter should only be called in DecodeHeaders

func (cb *filterManagerCallbackHandler) LookupConsumer(pluginName, key string) (api.Consumer, bool) {
	return consumer.LookupConsumer(cb.namespace, pluginName, key)
}

func (cb *filterManagerCallbackHandler) GetConsumer() api.Consumer {
	return cb.consumer
}

func (cb *filterManagerCallbackHandler) SetConsumer(c api.Consumer) {
	if c == nil {
		api.LogErrorf("set consumer with nil consumer: %s", debug.Stack())
		return
	}
	api.LogInfof("set consumer, namespace: %s, name: %s", cb.namespace, c.Name())
	cb.consumer = c
}

func (cb *filterManagerCallbackHandler) PluginState() api.PluginState {
	cb.cacheLock.Lock()
	if cb.pluginState == nil {
		cb.pluginState = pluginstate.NewPluginState()
	}
	cb.cacheLock.Unlock()
	return cb.pluginState
}
