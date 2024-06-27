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
	"regexp"

	"github.com/open-policy-agent/opa/rego"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
)

const (
	Name = "opa"
)

func init() {
	plugins.RegisterPluginType(Name, &Plugin{})
}

type Plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *Plugin) Type() plugins.PluginType {
	return plugins.TypeAuthz
}

func (p *Plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthz,
	}
}

func (p *Plugin) Config() api.PluginConfig {
	return &CustomConfig{}
}

type CustomConfig struct {
	Config
}

var (
	pkgMatcher = regexp.MustCompile(`^package\s+(\w+)\s`)
)

func (conf *CustomConfig) Validate() error {
	err := conf.Config.Validate()
	if err != nil {
		return err
	}

	local := conf.GetLocal()
	if local != nil {
		module := local.Text
		match := pkgMatcher.FindStringSubmatch(module)
		if len(match) < 2 {
			return errors.New("invalid Local.Text: bad package name")
		}
		policy := match[1]

		ctx := context.Background()

		_, err := rego.New(
			rego.Query(fmt.Sprintf("allow = data.%s.allow", policy)),
			rego.Module(fmt.Sprintf("%s.rego", policy), module),
		).PrepareForEval(ctx)

		if err != nil {
			return err
		}
	}
	return nil
}
