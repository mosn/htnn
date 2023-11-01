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
