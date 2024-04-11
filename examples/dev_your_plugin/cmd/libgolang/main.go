// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build so

// This package shows how to deploy your plugin in the data plane.
package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"mosn.io/htnn/api/pkg/filtermanager"
	_ "mosn.io/htnn/dev_your_plugin/plugins"
	// If you want to use the built-in plugins, you can import them here:
	// _ "mosn.io/htnn/plugins/plugins"
	//
	// Note that because we only update the module dependency during the release, if you use
	// a non-release version of mosn.io/htnn/xxx module, you may need to manually update the
	// dependency yourself, such as using replace in go.mod
)

func init() {
	http.RegisterHttpFilterConfigFactoryAndParser("fm", filtermanager.FilterManagerFactory, &filtermanager.FilterManagerConfigParser{})
}

func main() {}
