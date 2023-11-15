//go:build so

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"mosn.io/moe/pkg/filtermanager"
	_ "mosn.io/moe/plugins"
)

// Version is specified by build tag, in VERSION file
var (
	Version string = ""
)

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser("fm", filtermanager.FilterManagerConfigFactory, &filtermanager.FilterManagerConfigParser{})
}

func main() {}
