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

package opa

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/opa"
)

func TestOpaRemote(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	cli := http.DefaultClient
	f := factory(&config{
		CustomConfig: opa.CustomConfig{
			Config: opa.Config{
				ConfigType: &opa.Config_Remote{
					Remote: &opa.Remote{
						Url:    "http://127.0.0.1:8181",
						Policy: "httpapi/authz",
					},
				},
			},
		},
		client: cli,
	}, cb)
	hdr := envoy.NewRequestHeaderMap(http.Header(map[string][]string{
		":path": {"/?a=1&b&c=true&c=foo"},
		"pet":   {"cat"},
		"fruit": {"apple", "banana"},
	}))

	tests := []struct {
		name            string
		status          int
		checkInput      func(input map[string]interface{})
		resp            string
		respErr         error
		expectedMsg     string
		expectedHeaders http.Header
	}{
		{
			name: "happy path",
			resp: `{"result":{"allow":true}}`,
			checkInput: func(input map[string]interface{}) {
				assert.Equal(t, map[string]interface{}{
					"method": "GET",
					"scheme": "http",
					"host":   "localhost",
					"path":   "/",
					"query": map[string]interface{}{
						"a": "1",
						"b": "",
						"c": "true,foo",
					},
					"headers": map[string]interface{}{
						"pet":   "cat",
						"fruit": "apple,banana",
					},
				}, input["input"].(map[string]interface{})["request"])
			},
		},
		{
			name:   "reject",
			status: 403,
			resp:   `{"result":{"allow":false}}`,
		},
		{
			name:   "bad resp",
			status: 503,
			resp:   `{"result":{"`,
		},
		{
			name:    "bad resp2",
			status:  503,
			respErr: io.ErrUnexpectedEOF,
		},
		{
			name:   "custom response with status code and message",
			status: 401,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"status_code": 401,
						"body": "Authentication failed"
					}
				}
			}`,
			expectedMsg: "Authentication failed",
		},
		{
			name:   "custom response with headers",
			status: 429,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"status_code": 429,
						"body": "Rate limit exceeded",
						"headers": {
							"X-Rate-Limit": ["100"],
							"Retry-After": ["60"],
							"X-Request-ID": ["abc123"]
						}
					}
				}
			}`,
			expectedMsg: "Rate limit exceeded",
			expectedHeaders: http.Header{
				"X-Rate-Limit": {"100"},
				"Retry-After":  {"60"},
				"X-Request-Id": {"abc123"},
				"Content-Type": []string{"text/plain"},
			},
		},
		{
			name:   "custom response with complex headers",
			status: 422,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"status_code": 422,
						"body": "Validation failed",
						"headers": {
							"Content-Type": ["application/json"],
							"X-Validation-Errors": ["field1", "field2", "field3"],
							"X-Error-Code": ["VALIDATION_FAILED"]
						}
					}
				}
			}`,
			expectedMsg: "Validation failed",
			expectedHeaders: http.Header{
				"Content-Type":        {"application/json"},
				"X-Validation-Errors": {"field1", "field2", "field3"},
				"X-Error-Code":        {"VALIDATION_FAILED"},
			},
		},
		{
			name:   "custom response with status code only",
			status: 418,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"status_code": 418
					}
				}
			}`,
		},
		{
			name:   "custom response with message only",
			status: 403,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"body": "Access denied"
					}
				}
			}`,
			expectedMsg: "Access denied",
		},
		{
			name:   "custom response with empty object",
			status: 403,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {}
				}
			}`,
		},
		{
			name:   "custom response with null value",
			status: 403,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": null
				}
			}`,
		},
		{
			name: "allow true ignores custom response",
			resp: `{
				"result": {
					"allow": true,
					"custom_response": {
						"status_code": 401,
						"body": "This message should be ignored"
					}
				}
			}`,
		},
		{
			name:   "custom response for service unavailable",
			status: 503,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"status_code": 503,
						"body": "Service temporarily unavailable",
						"headers": {
							"Retry-After": ["300"],
							"X-Service-Status": ["maintenance"],
							"X-Maintenance-Window": ["2025-06-05T06:00:00Z", "2025-06-05T08:00:00Z"]
						}
					}
				}
			}`,
			expectedMsg: "Service temporarily unavailable",
			expectedHeaders: http.Header{
				"Retry-After":          {"300"},
				"X-Service-Status":     {"maintenance"},
				"Content-Type":         []string{"text/plain"},
				"X-Maintenance-Window": {"2025-06-05T06:00:00Z", "2025-06-05T08:00:00Z"},
			},
		},
		{
			name:   "custom response with zero status code",
			status: 403,
			resp: `{
				"result": {
					"allow": false,
					"custom_response": {
						"status_code": 0,
						"body": "Zero status code test"
					}
				}
			}`,
			expectedMsg: "Zero status code test",
		},
		{
			name:   "custom response not present, deny by default",
			resp:   `{"result":{"allow":false}}`,
			status: 403,
		},
		{
			name: "header array contains non-string, should be ignored",
			resp: `{
		"result": {
			"allow": false,
			"custom_response": {
				"status_code": 400,
				"body": "bad array",
				"headers": {
					"X-Bad": ["ok", "123"]
				}
			}
		}
	}`,
			status:      400,
			expectedMsg: "bad array",
			expectedHeaders: http.Header{
				"X-Bad":        {"ok", "123"},
				"Content-Type": {"text/plain"},
			},
		},
		{
			name: "header value is single string",
			resp: `{
		"result": {
			"allow": false,
			"custom_response": {
				"status_code": 401,
				"body": "single value",
				"headers": {
					"X-Note": ["just one"]
				}
			}
		}
	}`,
			status:      401,
			expectedMsg: "single value",
			expectedHeaders: http.Header{
				"X-Note":       {"just one"},
				"Content-Type": {"text/plain"},
			},
		},
		{
			name:   "custom response with no Content-Type header, should fallback to text/plain",
			status: 401,
			resp: `{
		"result": {
			"allow": false,
			"custom_response": {
				"status_code": 401,
				"body": "No content-type header present"
			}
		}
	}`,
			expectedMsg: "No content-type header present",
			expectedHeaders: http.Header{
				"Content-Type": {"text/plain"},
			},
		},
		{
			name:   "custom response with lowercase content-type header",
			status: 401,
			resp: `{
		"result": {
			"allow": false,
			"custom_response": {
				"status_code": 401,
				"body": "Lowercase header test",
				"headers": {
					"content-type": ["application/json"]
				}
			}
		}
	}`,
			expectedMsg: "Lowercase header test",
			expectedHeaders: http.Header{
				"Content-Type": {"application/json"},
			},
		},
		{
			name:   "custom response with empty content-type header value, fallback expected",
			status: 401,
			resp: `{
		"result": {
			"allow": false,
			"custom_response": {
				"status_code": 401,
				"body": "Empty content-type",
				"headers": {
					"Content-Type": []
				}
			}
		}
	}`,
			expectedMsg: "Empty content-type",
			expectedHeaders: http.Header{
				"Content-Type": {"text/plain"},
			},
		},
		{
			name:   "custom response with correct application/json content-type",
			status: 403,
			resp: `{
		"result": {
			"allow": false,
			"custom_response": {
				"status_code": 403,
				"body": "Expected JSON",
				"headers": {
					"Content-Type": ["application/json"],
					"X-Debug": ["ok"]
				}
			}
		}
	}`,
			expectedMsg: "Expected JSON",
			expectedHeaders: http.Header{
				"Content-Type": {"application/json"},
				"X-Debug":      {"ok"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{}
			resp.Body = io.NopCloser(bytes.NewReader([]byte(tt.resp)))
			patches := gomonkey.ApplyMethodFunc(cli, "Post",
				func(url, contentType string, body io.Reader) (*http.Response, error) {
					if tt.checkInput != nil {
						input := map[string]interface{}{}
						data, _ := io.ReadAll(body)
						_ = json.Unmarshal(data, &input)
						tt.checkInput(input)
					}
					return resp, tt.respErr
				})
			defer patches.Reset()

			lr, ok := f.DecodeHeaders(hdr, true).(*api.LocalResponse)
			if !ok {
				assert.Equal(t, tt.status, 0)
			} else {
				assert.Equal(t, tt.status, lr.Code, "Status code mismatch")

				if tt.expectedMsg != "" {
					assert.Equal(t, tt.expectedMsg, lr.Msg, "Message mismatch")
				}

				if tt.expectedHeaders != nil {
					assert.Equal(t, tt.expectedHeaders, lr.Header, "Headers mismatch")
				}
			}
		})
	}
}

func TestOpaLocal(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	hdr := envoy.NewRequestHeaderMap(http.Header(map[string][]string{
		":path": {"/?a=1&b&c=true&c=foo"},
		"fruit": {"apple", "banana"},
	}))

	tests := []struct {
		name            string
		status          int
		text            string
		expectedMsg     string
		expectedHeaders http.Header
	}{
		{
			name: "happy path",
			text: `default allow = true`,
		},
		{
			name: "check input",
			text: `import input.request
				import future.keywords
				default allow = false
				allow {
					request.method == "GET"
					request.path == "/"
					startswith(request.headers.fruit, "apple")
					startswith(request.query.c, "true")
				}`,
		},
		{
			name: "reject",
			text: `import input.request
				import future.keywords
				default allow = false
				allow {
					endswith(request.query.c, "true")
				}`,
			status: 403,
		},
		{
			name:   "bad result",
			text:   `import input.request`,
			status: 503,
		},
		{
			name:   "no bool result",
			text:   `default allow = "a"`,
			status: 503,
		},
		{
			name: "custom response with status and message",
			text: `import input.request
				import future.keywords
				default allow = false
				default custom_response = {
					"status_code": 401,
					"body": "Unauthorized access"
				}
				allow {
					false
				}`,
			status:      401,
			expectedMsg: "Unauthorized access",
		},
		{
			name: "custom response with headers",
			text: `import input.request
				import future.keywords
				default allow = false
				default custom_response = {
					"status_code": 429,
					"body": "Rate limit exceeded",
					"headers": {
						"X-Rate-Limit": ["100"],
						"Retry-After": ["60"]
					}
				}
				allow {
					false
				}`,
			status:      429,
			expectedMsg: "Rate limit exceeded",
			expectedHeaders: http.Header{
				"X-Rate-Limit": {"100"},
				"Retry-After":  {"60"},
				"Content-Type": {"text/plain"},
			},
		},
		{
			name: "custom response with multiple header values",
			text: `import input.request
				import future.keywords
				default allow = false
				default custom_response = {
					"status_code": 422,
					"body": "Validation failed",
					"headers": {
						"X-Error": ["field1", "field2"],
						"X-Request-ID": ["12345"]
					}
				}
				allow {
					false
				}`,
			status:      422,
			expectedMsg: "Validation failed",
			expectedHeaders: http.Header{
				"X-Error":      {"field1", "field2"},
				"X-Request-Id": {"12345"},
				"Content-Type": {"text/plain"},
			},
		},
		{
			name: "custom response with zero status code should default to 403",
			text: `import input.request
				import future.keywords
				default allow = false
				default custom_response = {
					"status_code": 0,
					"body": "Default status"
				}
				allow {
					false
				}`,
			status:      403,
			expectedMsg: "Default status",
		},
		{
			name: "custom response without status code",
			text: `import input.request
				import future.keywords
				default allow = false
				default custom_response = {
					"body": "No status code provided"
				}
				allow {
					false
				}`,
			status:      403,
			expectedMsg: "No status code provided",
		},
		{
			name: "allow true with custom response should be ignored",
			text: `import input.request
				import future.keywords
				default allow = true
				default custom_response = {
					"status_code": 401,
					"body": "This should be ignored"
				}
				allow {
					true
				}`,
		},
		{
			name: "custom response not provided, default to 403",
			text: `import input.request
		default allow = false
		allow {
			false
		}`,
			status: 403,
		},
		{
			name: "header array contains non-string, should be ignored",
			text: `import input.request
		import future.keywords
		default allow = false
		default custom_response = {
			"status_code": 400,
			"body": "mixed header value types",
			"headers": {
				"X-Bad": ["ok", 123]
			}
		}
		allow {
			false
		}`,
			status:      400,
			expectedMsg: "mixed header value types",
			expectedHeaders: http.Header{
				"Content-Type": {"text/plain"},
			},
		},
		{
			name: "custom response with lowercase content-type",
			text: `import input.request
		import future.keywords
		default allow = false
		default custom_response = {
			"status_code": 401,
			"body": "invalid",
			"headers": {
				"content-type": ["application/json"]
			}
		}
		allow {
			false
		}`,
			status:      401,
			expectedMsg: "invalid",
			expectedHeaders: http.Header{
				"Content-Type": {"application/json"},
			},
		},
		{
			name: "custom response without content-type header, should default to text/plain",
			text: `import input.request
		import future.keywords
		default allow = false
		default custom_response = {
			"status_code": 401,
			"body": "invalid"
		}
		allow {
			false
		}`,
			status:      401,
			expectedMsg: "invalid",
			expectedHeaders: http.Header{
				"Content-Type": {"text/plain"},
			},
		},
		{
			name: "custom response with empty content-type array, should fallback to text/plain",
			text: `import input.request
		import future.keywords
		default allow = false
		default custom_response = {
			"status_code": 401,
			"body": "invalid",
			"headers": {
				"Content-Type": []
			}
		}
		allow {
			false
		}`,
			status:      401,
			expectedMsg: "invalid",
			expectedHeaders: http.Header{
				"Content-Type": {"text/plain"},
			},
		},
		{
			name: "custom response with proper Content-Type should not be overridden",
			text: `import input.request
		import future.keywords
		default allow = false
		default custom_response = {
			"status_code": 401,
			"body": "invalid",
			"headers": {
				"Content-Type": ["application/json"]
			}
		}
		allow {
			false
		}`,
			status:      401,
			expectedMsg: "invalid",
			expectedHeaders: http.Header{
				"Content-Type": {"application/json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &config{
				CustomConfig: opa.CustomConfig{
					Config: opa.Config{
						ConfigType: &opa.Config_Local{
							Local: &opa.Local{
								Text: "package test\n" + tt.text,
							},
						},
					},
				},
			}
			err := c.Init(nil)
			require.NoError(t, err)
			f := factory(c, cb)
			lr, ok := f.DecodeHeaders(hdr, true).(*api.LocalResponse)
			if !ok {
				assert.Equal(t, tt.status, 0)
			} else {
				assert.Equal(t, tt.status, lr.Code, "CustomResponse Status code mismatch")

				if tt.expectedMsg != "" {
					assert.Equal(t, tt.expectedMsg, lr.Msg, "CustomResponse Message mismatch")
				}

				if tt.expectedHeaders != nil {
					assert.Equal(t, tt.expectedHeaders, lr.Header, "CustomResponse Headers mismatch")
				}
			}
		})
	}
}
