//go:build so

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/pkg/plugins"
	_ "mosn.io/moe/plugins"
	_ "mosn.io/moe/tests/integration/plugins"
)

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser("fm", filtermanager.FilterManagerConfigFactory, &filtermanager.FilterManagerConfigParser{})
	plugins.IterateHttpPlugin(func(name string, plugin plugins.Plugin) bool {
		filtermanager.RegisterHttpFilterConfigFactoryAndParser(name, plugin.ConfigFactory(), plugin.ConfigParser())
		return true
	})
}

func main() {}
