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

package oidc

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/avast/retry-go"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/plugins"
)

const (
	Name = "oidc"
)

func init() {
	plugins.RegisterHttpPlugin(Name, &plugin{})
}

type plugin struct {
	plugins.PluginMethodDefaultImpl
}

func (p *plugin) Type() plugins.PluginType {
	return plugins.TypeAuthn
}

func (p *plugin) Order() plugins.PluginOrder {
	return plugins.PluginOrder{
		Position: plugins.OrderPositionAuthn,
	}
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	Config

	opTimeout      time.Duration
	oauth2Config   *oauth2.Config
	verifier       *oidc.IDTokenVerifier
	cookieEncoding *securecookie.SecureCookie
	refreshLeeway  time.Duration
	cookieEntryID  string
}

func (conf *config) ctxWithClient(ctx context.Context) context.Context {
	httpClient := &http.Client{Timeout: conf.opTimeout}
	return context.WithValue(ctx, oauth2.HTTPClient, httpClient)
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	if conf.IdTokenHeader == "" {
		conf.IdTokenHeader = "x-id-token"
	}

	du := 3 * time.Second
	timeout := conf.GetTimeout()
	if timeout != nil {
		du = timeout.AsDuration()
	}
	conf.opTimeout = du

	du = 10 * time.Second
	leeway := conf.GetAccessTokenRefreshLeeway()
	if leeway != nil {
		du = leeway.AsDuration()
	}
	conf.refreshLeeway = du

	ctx := conf.ctxWithClient(context.Background())
	var provider *oidc.Provider
	var err error
	err = retry.Do(
		func() error {
			provider, err = oidc.NewProvider(ctx, conf.Issuer)
			return err
		},
		retry.RetryIf(func(err error) bool {
			api.LogWarnf("failed to get oidc provider, err: %v", err)
			return true
		}),
		retry.Attempts(3),
		// backoff delay
		retry.Delay(500*time.Millisecond),
	)
	if err != nil {
		return err
	}

	if !conf.DisableAccessTokenRefresh {
		conf.Scopes = append(conf.Scopes, oidc.ScopeOfflineAccess)
	}
	conf.oauth2Config = &oauth2.Config{
		ClientID:     conf.ClientId,
		ClientSecret: conf.ClientSecret,
		// ScopeOpenID is the mandatory scope for all OpenID Connect OAuth2 requests.
		Scopes:      append([]string{oidc.ScopeOpenID}, conf.Scopes...),
		RedirectURL: conf.RedirectUrl,

		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),
	}
	conf.verifier = provider.Verifier(&oidc.Config{ClientID: conf.ClientId})
	conf.cookieEncoding = securecookie.New([]byte(conf.ClientSecret), nil)
	conf.cookieEntryID = base64.RawURLEncoding.EncodeToString([]byte(conf.ClientId))
	return nil
}
