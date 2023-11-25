package request

import (
	"net/url"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

func GetUrl(header api.RequestHeaderMap) *url.URL {
	path := header.Path()
	uri, err := url.ParseRequestURI(path)
	if err != nil {
		api.LogErrorf("failed to parse uri: %v", err)
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
