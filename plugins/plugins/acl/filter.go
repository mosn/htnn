package acl

import (
    "net"
    "mosn.io/htnn/api/pkg/filtermanager/api"
    "mosn.io/htnn/types/plugins/acl"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
    return &filter{
        config:    c.(*acl.Config),
        callbacks: callbacks,
    }
}

type filter struct {
    api.PassThroughFilter

    config    *acl.Config
    callbacks api.FilterCallbackHandler
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
    clientIP, ok := headers.Get("X-Forwarded-For")
    if !ok {
        return &api.LocalResponse{Code: 403, Msg: "client IP not found"}
    }

    for _, denyIP := range f.config.DenyList {
        if isIPMatch(clientIP, denyIP) {
            return &api.LocalResponse{Code: 403, Msg: "access denied"}
        }
    }

    for _, allowIP := range f.config.AllowList {
        if isIPMatch(clientIP, allowIP) {
            return api.Continue
        }
    }

    return &api.LocalResponse{Code: 403, Msg: "access denied"}
}

func isIPMatch(clientIP, ruleIP string) bool {
    client := net.ParseIP(clientIP)
    _, cidr, err := net.ParseCIDR(ruleIP)
    if err != nil {
        return client.String() == ruleIP
    }
    return cidr.Contains(client)
}