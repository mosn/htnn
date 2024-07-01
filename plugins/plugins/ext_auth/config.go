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

package ext_auth

import (
	"net/http"
	"time"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/types/pkg/expr"
	"mosn.io/htnn/types/plugins/ext_auth"
)

func init() {
	plugins.RegisterPlugin(ext_auth.Name, &plugin{})
}

type plugin struct {
	ext_auth.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	ext_auth.Config

	client                  *http.Client
	headerToUpstreamMatcher expr.Matcher
	headerToClientMatcher   expr.Matcher
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	du := 200 * time.Millisecond
	timeout := conf.GetHttpService().GetTimeout()
	if timeout != nil {
		du = timeout.AsDuration()
	}

	conf.client = &http.Client{Timeout: du}

	resp := conf.GetHttpService().GetAuthorizationResponse()
	if resp != nil {
		if len(resp.AllowedUpstreamHeaders) > 0 {
			matcher, err := expr.BuildRepeatedStringMatcherIgnoreCase(resp.AllowedUpstreamHeaders)
			if err != nil {
				return err
			}
			conf.headerToUpstreamMatcher = matcher
		}
		if len(resp.AllowedClientHeaders) > 0 {
			matcher, err := expr.BuildRepeatedStringMatcherIgnoreCase(resp.AllowedClientHeaders)
			if err != nil {
				return err
			}
			conf.headerToClientMatcher = matcher
		}
	}
	return nil
}
