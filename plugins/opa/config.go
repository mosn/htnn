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
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/open-policy-agent/opa/rego"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

const (
	Name = "opa"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeAuthz
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthz,
	}
}

func (p *plugin) ConfigFactory() api.FilterConfigFactory {
	return configFactory
}

func (p *plugin) Config() plugins.PluginConfig {
	return &config{}
}

type config struct {
	Config

	client *http.Client
	query  rego.PreparedEvalQuery
}

var (
	pkgMatcher = regexp.MustCompile(`^package\s+(\w+)\s`)
)

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	remote := conf.GetRemote()
	if remote != nil {
		conf.client = &http.Client{Timeout: 200 * time.Millisecond}
		return nil
	}

	local := conf.GetLocal()
	module := local.Text
	match := pkgMatcher.FindStringSubmatch(module)
	if len(match) < 2 {
		return errors.New("invalid Local.Text: bad package name")
	}
	policy := match[1]

	ctx := context.Background()

	query, err := rego.New(
		rego.Query(fmt.Sprintf("allow = data.%s.allow", policy)),
		rego.Module(fmt.Sprintf("%s.rego", policy), module),
	).PrepareForEval(ctx)

	if err != nil {
		return err
	}

	conf.query = query
	return nil
}
