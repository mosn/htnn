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

//go:build so && !envoy1.29

package main

import (
	"github.com/envoyproxy/envoy/contrib/golang/filters/http/source/go/pkg/http"

	"mosn.io/htnn/api/pkg/consumer"
	"mosn.io/htnn/api/pkg/dynamicconfig"
	"mosn.io/htnn/api/pkg/filtermanager"
	_ "mosn.io/htnn/api/plugins/tests/integration/dataplane" // for utility plugins provided in the test framework
	_ "mosn.io/htnn/api/tests/integration"                   // for plugins used in the test
)

func init() {
	http.RegisterHttpFilterFactoryAndConfigParser("fm", filtermanager.FilterManagerFactory, &filtermanager.FilterManagerConfigParser{})
	http.RegisterHttpFilterFactoryAndConfigParser("fm-metrics", filtermanager.MetricsConfigFactory, &filtermanager.MetricsConfigParser{})
	http.RegisterHttpFilterFactoryAndConfigParser("cm", consumer.ConsumerManagerFactory, &consumer.ConsumerManagerConfigParser{})
	http.RegisterHttpFilterFactoryAndConfigParser("dc", dynamicconfig.DynamicConfigFactory, &dynamicconfig.DynamicConfigParser{})
}

func main() {}
