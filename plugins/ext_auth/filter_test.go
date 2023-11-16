package ext_auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	"mosn.io/moe/pkg/filtermanager/api"
	"mosn.io/moe/tests/pkg/envoy"
)

func response(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       http.NoBody,
	}
}

func TestExtAuth(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		hdr    map[string][]string
		server func(r *http.Request) (*http.Response, error)
		res    api.ResultAction
	}{
		{
			name: "default",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth"
			}}`,
			hdr: map[string][]string{
				"Authorization": {"Basic amFjazIwMjE6MTIzNDU2"},
				"Other":         {"not passed"},
			},
			server: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "test.local", r.Host)
				assert.Equal(t, "/ext_auth/", r.URL.Path)
				assert.Equal(t, "Basic amFjazIwMjE6MTIzNDU2", r.Header.Get("Authorization"))
				assert.Equal(t, "", r.Header.Get("Other"))
				return response(200), nil
			},
		},
		{
			name: "add headers",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth",
				"authorization_request": {
					"headers_to_add": [
						{"key": "foo", "value": "bar"},
						{"key": "foo", "value": "baz"}
					]
				}
			}}`,
			hdr: map[string][]string{
				"Foo": {"blah"},
			},
			server: func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, []string{"baz"}, r.Header.Values("Foo"))
				return response(200), nil
			},
		},
		{
			name: "auth denied",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth"
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				return response(401), nil
			},
			res: &api.LocalResponse{Code: 401},
		},
		{
			name: "auth error",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth"
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("ouch")
			},
			res: &api.LocalResponse{Code: 403},
		},
		{
			name: "auth error, status_on_error configured",
			input: `{"http_service":{
				"url": "http://127.0.0.1:10001/ext_auth",
				"status_on_error": 401
			}}`,
			server: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("ouch")
			},
			res: &api.LocalResponse{Code: 401},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := envoy.NewFilterCallbackHandler()
			conf := &config{}
			protojson.Unmarshal([]byte(tt.input), conf)
			conf.Init(nil)
			patches := gomonkey.ApplyMethodFunc(conf.client, "Do", tt.server)
			defer patches.Reset()
			f := configFactory(conf)(cb)
			defaultHdr := map[string][]string{
				":authority": {"test.local"},
				":method":    {"DELETE"},
				":path":      {"/"},
			}
			for k, v := range tt.hdr {
				defaultHdr[k] = v
			}
			hdr := envoy.NewRequestHeaderMap(http.Header(defaultHdr))
			assert.Equal(t, tt.res, f.DecodeHeaders(hdr, true))
		})
	}
}
