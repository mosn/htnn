//go:build so

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	_ "mosn.io/moe/dev_your_plugin/plugins"
	"mosn.io/moe/pkg/plugins"
	_ "mosn.io/moe/plugins"
)

func init() {
	plugins.IterateHttpPlugin(func(name string, plugin plugins.Plugin) bool {
		api.LogWarnf("register plugin %s", name)
		http.RegisterHttpFilterConfigFactoryAndParser(name, plugin.ConfigFactory(), plugin.ConfigParser())
		return true
	})
}

func main() {}
