//go:build so

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	_ "mosn.io/moe/dev_your_plugin/plugins"
	"mosn.io/moe/pkg/filtermanager"
	_ "mosn.io/moe/plugins"
)

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser("fm", filtermanager.FilterManagerConfigFactory, &filtermanager.FilterManagerConfigParser{})
}

func main() {}
