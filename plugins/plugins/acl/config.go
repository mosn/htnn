package acl

import (
    "mosn.io/htnn/api/pkg/filtermanager/api"
    "mosn.io/htnn/api/pkg/plugins"
    "mosn.io/htnn/types/plugins/acl"
)

func init() {
    plugins.RegisterPlugin(acl.Name, &plugin{})
}

type plugin struct {
    acl.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
    return factory
}