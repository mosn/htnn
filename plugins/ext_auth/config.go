package ext_auth

import (
	"net/http"
	"time"

	"mosn.io/moe/pkg/expr"
	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/pkg/plugins"
)

const (
	// We name this plugin as ext_auth to distinguish it from the C++ implementation ext_authz.
	// We may add new feature to this plugin which will make it different from its C++ sibling.
	Name = "ext_auth"
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
