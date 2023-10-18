//go:build so

package demo

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"
)

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser("demo", configFactory, &parser{})
}
