package basicauth

import (
    "encoding/base64"
    "strings"

    "mosn.io/htnn/api/pkg/filtermanager/api"
    "mosn.io/htnn/types/plugins/basicauth"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
    return &filter{
        config:    c.(*basicauth.Config),
        callbacks: callbacks,
    }
}

type filter struct {
    api.PassThroughFilter

    config    *basicauth.Config
    callbacks api.FilterCallbackHandler
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
    authHeader, ok := headers.Get("Authorization")
    if !ok || !strings.HasPrefix(authHeader, "Basic ") {
        return &api.LocalResponse{Code: 401, Msg: "missing or invalid Authorization header"}
    }

    encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
    decoded, err := base64.StdEncoding.DecodeString(encodedCredentials)
    if err != nil {
        return &api.LocalResponse{Code: 401, Msg: "invalid Authorization header"}
    }

    parts := strings.SplitN(string(decoded), ":", 2)
    if len(parts) != 2 {
        return &api.LocalResponse{Code: 401, Msg: "invalid Authorization header"}
    }

    username, password := parts[0], parts[1]
    if validPassword, ok := f.config.Credentials[username]; !ok || validPassword != password {
        return &api.LocalResponse{Code: 401, Msg: "invalid username or password"}
    }

    return api.Continue
}