package basicauth

import (
    "mosn.io/htnn/api/pkg/filtermanager/api"
    "mosn.io/htnn/api/pkg/plugins"
    "mosn.io/htnn/types/plugins/basicauth"
)

func init() {
    plugins.RegisterPlugin(basicauth.Name, &plugin{})
}

type plugin struct {
    basicauth.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
    return factory
}