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
