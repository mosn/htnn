package basicauth

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/basicauth"
)

func TestBasicAuthFilter(t *testing.T) {
    tests := []struct {
        name     string
        conf     string
        headers  map[string][]string
        expected int
    }{
        {
            name: "valid credentials",
            conf: `{
                "credentials": {
                    "user1": "password1",
                    "user2": "password2"
                }
            }`,
        	headers: map[string][]string{
                "Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("user1:password1"))},
            },
            expected: 0,
        },
        {
            name: "invalid credentials",
            conf: `{
                "credentials": {
                    "user1": "password1",
                    "user2": "password2"
                }
            }`,
            headers: map[string][]string{
                "Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("user1:wrongpassword"))},
            },
            expected: 401,
        },
        {
            name: "missing Authorization header",
            conf: `{
                "credentials": {
                    "user1": "password1",
                    "user2": "password2"
                }
            }`,
            headers: map[string][]string{},
            expected: 401,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            conf := &basicauth.Config{}
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