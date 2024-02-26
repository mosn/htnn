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
	"net/http"
	"net/textproto"
	"strings"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"golang.org/x/net/http/httpguts"
)

// The cookie parser is from Go's http/cookie.go, which are not exported

func isNotToken(r rune) bool {
	return !httpguts.IsTokenRune(r)
}

func isCookieNameValid(raw string) bool {
	if raw == "" {
		return false
	}
	return strings.IndexFunc(raw, isNotToken) < 0
}

func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}

func parseCookieValue(raw string, allowDoubleQuote bool) (string, bool) {
	// Strip the quotes, if present.
	if allowDoubleQuote && len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	for i := 0; i < len(raw); i++ {
		if !validCookieValueByte(raw[i]) {
			return "", false
		}
	}
	return raw, true
}

// If multiple cookies match the given name, only one cookie will be returned.
func ParseCookies(headers api.RequestHeaderMap) map[string]*http.Cookie {
	lines := headers.Values("Cookie")
	if len(lines) == 0 {
		return map[string]*http.Cookie{}
	}

	cookies := make(map[string]*http.Cookie, len(lines)+strings.Count(lines[0], ";"))
	for _, line := range lines {
		line = textproto.TrimString(line)

		var part string
		for len(line) > 0 { // continue since we have rest
			part, line, _ = strings.Cut(line, ";")
			part = textproto.TrimString(part)
			if part == "" {
				continue
			}
			name, val, _ := strings.Cut(part, "=")
			name = textproto.TrimString(name)
			if !isCookieNameValid(name) {
				continue
			}
			val, ok := parseCookieValue(val, true)
			if !ok {
				continue
			}
			cookies[name] = &http.Cookie{Name: name, Value: val}
		}
	}
	return cookies
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
