//go:build so

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
	_ "mosn.io/moe/plugins"
)

// Version is specified by build tag, in VERSION file
var (
	Version string = ""
)

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser("fm", filtermanager.FilterManagerConfigFactory, &filtermanager.FilterManagerConfigParser{})
	plugins.IterateHttpPlugin(func(name string, plugin plugins.Plugin) bool {
		api.LogWarnf("register plugin %s", name)
		filtermanager.RegisterHttpFilterConfigFactoryAndParser(name, plugin.ConfigFactory(), plugin.ConfigParser())
		return true
	})
}

func main() {}
