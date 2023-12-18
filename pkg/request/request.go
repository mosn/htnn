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

package request

import (
	"fmt"
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

func GetUrl(header api.RequestHeaderMap) *url.URL {
	path := header.Path()
	uri, err := url.ParseRequestURI(path)
	if err != nil {
		panic(fmt.Sprintf("unexpected bad request uri given by envoy: %v", err))
	}
	return uri
}

// GetHeaders returns a plain map represents the headers. The returned headers won't
// contain any pseudo header like `:authority`.
func GetHeaders(header api.RequestHeaderMap) map[string][]string {
	hdr := map[string][]string{}
	header.Range(func(k, v string) bool {
		if k[0] == ':' {
			return true
		}
		if entry, ok := hdr[k]; !ok {
			hdr[k] = []string{v}
		} else {
			hdr[k] = append(entry, v)
		}
		return true
	})
	return hdr
}
