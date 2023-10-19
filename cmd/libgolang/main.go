//go:build so

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"mosn.io/moe/pkg/plugins"
	_ "mosn.io/moe/plugins"
)

// Version is specified by build tag, in VERSION file
var (
	Version string = ""
)

func init() {
	plugins.IterateHttpPlugin(func(name string, plugin plugins.Plugin) bool {
		http.RegisterHttpFilterConfigFactoryAndParser(name, plugin.ConfigFactory(), plugin.ConfigParser())
		return true
	})
}

func main() {}
