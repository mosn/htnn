package acl

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/acl"
)

func TestACLFilter(t *testing.T) {
    tests := []struct {
        name     string
        conf     string
        headers  map[string][]string
        expected int
    }{
        {
            name: "deny list match",
            conf: `{
                "allow_list": ["192.168.1.0/24"],
                "deny_list": ["10.0.0.1"]
            }`,
            headers: map[string][]string{
                "X-Forwarded-For": {"10.0.0.1"},
            },
            expected: 403,
        },
        {
            name: "allow list match",
            conf: `{
                "allow_list": ["192.168.1.0/24"],
                "deny_list": ["10.0.0.1"]
            }`,
            headers: map[string][]string{
                "X-Forwarded-For": {"192.168.1.50"},
            },
            expected: 0,
        },
        {
            name: "no match",
            conf: `{
                "allow_list": ["192.168.1.0/24"],
                "deny_list": ["10.0.0.1"]
            }`,
            headers: map[string][]string{
                "X-Forwarded-For": {"8.8.8.8"},
            },
            expected: 403,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            conf := &acl.Config{}
            err := protojson.Unmarshal([]byte(tt.conf), conf)
            require.NoError(t, err)

            cb := envoy.NewFilterCallbackHandler()
            f := factory(conf, cb)

            httpHdr := http.Header{}
            for k, v := range tt.headers {
                for _, vv := range v {
                    httpHdr.Add(k, vv)
                }
            }
            hdr := envoy.NewRequestHeaderMap(httpHdr)

            res := f.DecodeHeaders(hdr, true)
            if tt.expected != 0 {
                r, ok := res.(*api.LocalResponse)
                require.True(t, ok)
                assert.Equal(t, tt.expected, r.Code)
            } else {
                assert.Equal(t, api.Continue, res)
            }
        })
    }
}