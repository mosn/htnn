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

package opa

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/open-policy-agent/opa/rego"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/plugins/opa"
)

func init() {
	plugins.RegisterPlugin(opa.Name, &plugin{})
}

type plugin struct {
	opa.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	opa.CustomConfig

	client *http.Client
	query  rego.PreparedEvalQuery
}

var (
	pkgMatcher = regexp.MustCompile(`^package\s+(\w+)\s`)
)

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	remote := conf.GetRemote()
	if remote != nil {
		var timeout time.Duration
		if remote.Timeout != nil {
			timeout = remote.Timeout.AsDuration()
		} else {
			timeout = 200 * time.Millisecond
		}
		conf.client = &http.Client{Timeout: timeout}
		return nil
	}

	local := conf.GetLocal()
	module := local.Text
	match := pkgMatcher.FindStringSubmatch(module)
	policy := match[1]

	ctx := context.Background()

	query, _ := rego.New(
		rego.Query(fmt.Sprintf(`
            allow = data.%[1]s.allow;
            custom_response = object.get(data.%[1]s, "custom_response", null)
        `, policy)),
		rego.Module(fmt.Sprintf("%s.rego", policy), module),
	).PrepareForEval(ctx)
	conf.query = query
	return nil
}
