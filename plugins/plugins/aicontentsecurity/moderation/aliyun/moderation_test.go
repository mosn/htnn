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

package aliyun

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func TestRequestModeration_Live(t *testing.T) {
	t.Skip("Skipping live test to avoid dependency on external services")
	accessKeyID := ""
	accessKeySecret := ""

	conf := &aicontentsecurity.AliyunConfig{
		AccessKeyId:     accessKeyID,
		AccessKeySecret: accessKeySecret,
		Region:          "cn-shanghai",
		MaxRiskLevel:    "high",
	}
	mod, err := New(&aicontentsecurity.Config_AliyunConfig{AliyunConfig: conf})
	assert.NoError(t, err)
	m, ok := mod.(*Moderator)
	assert.True(t, ok, "m should be of type *Moderator")

	t.Run("live test with clean text", func(t *testing.T) {
		content := "Hello, this is a normal test text for the Aliyun content security API."
		result, err := m.Request(context.Background(), content, nil)
		assert.NoError(t, err)
		assert.True(t, result.Allow, "clean text should be allowed")
		t.Log("Clean text test passed, Allow=true")
	})

	t.Run("live test with dangerous text", func(t *testing.T) {
		content := "Using drugs" // Example of violative word
		result, err := m.Request(context.Background(), content, nil)
		assert.NoError(t, err)
		assert.False(t, result.Allow, "violative text should be rejected")
		t.Logf("Violative text test rejected, Allow=false, Reason: %s", result.Reason)
	})
}

// TestPercentEncode validates the custom percent encoding function
func TestPercentEncode(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"no special chars", "abc", "abc"},
		{"with space", "a b", "a%20b"},
		{"with plus", "a+b", "a%2Bb"}, // url.QueryEscape turns + into %2B
		{"with star", "a*b", "a%2Ab"},
		{"with tilde", "a~b", "a~b"}, // url.QueryEscape does not escape ~, but our code replaces %7E with ~
		{"complex string", "/:=?&", "%2F%3A%3D%3F%26"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, percentEncode(tc.input))
		})
	}
}

// TestGenerateSignature validates that the signature generation logic is correct
func TestGenerateSignature(t *testing.T) {
	m := &Moderator{
		accessKeySecret: "testSecret",
	}
	params := map[string]string{
		"Format":            "JSON",
		"Version":           "2022-03-02",
		"AccessKeyId":       "testId",
		"SignatureMethod":   "HMAC-SHA1",
		"Timestamp":         "2025-08-03T12:00:00Z",
		"SignatureVersion":  "1.0",
		"SignatureNonce":    "a-unique-nonce",
		"Action":            "TextModerationPlus",
		"Service":           "llm_query_moderation",
		"ServiceParameters": `{"content":"test"}`,
	}

	// Pre-calculated correct signature
	expectedSignature := "UU3dTtATew0t/yqVOkhtOju3Dfg="
	actualSignature := m.generateSignature(params)
	assert.Equal(t, expectedSignature, actualSignature)
}

func TestModerator_Call_WithGomonkey(t *testing.T) {
	conf := &aicontentsecurity.AliyunConfig{AccessKeyId: "id", AccessKeySecret: "secret"}
	mod, err := New(&aicontentsecurity.Config_AliyunConfig{AliyunConfig: conf})
	assert.NoError(t, err)
	m, ok := mod.(*Moderator)
	assert.True(t, ok, "m should be of type *Moderator")

	t.Run("should succeed when http call is successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			respBody := `{"Code":200}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(respBody)),
			}, nil
		})

		body, err := m.call(context.Background(), "service", "params")
		assert.NoError(t, err)
		assert.Equal(t, `{"Code":200}`, string(body))
	})

	t.Run("should fail when http client returns an error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		expectedErr := errors.New("network error")
		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			return nil, expectedErr
		})

		_, err := m.call(context.Background(), "service", "params")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("should fail when status code is not 200", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("server error")),
			}, nil
		})

		_, err := m.call(context.Background(), "service", "params")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server error")
	})

	t.Run("should fail when reading response body fails", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(&errReader{}),
			}, nil
		})

		_, err := m.call(context.Background(), "service", "params")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read response")
	})
}

type errReader struct{}

func (er *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("i/o error")
}

func TestModerator_ExecuteModerationService_WithGomonkey(t *testing.T) {
	conf := &aicontentsecurity.AliyunConfig{AccessKeyId: "id", AccessKeySecret: "secret"}
	mod, err := New(&aicontentsecurity.Config_AliyunConfig{AliyunConfig: conf})
	assert.NoError(t, err)
	m, ok := mod.(*Moderator)
	assert.True(t, ok, "m should be of type *Moderator")

	t.Run("should succeed when http call is successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		respBody := `{"Code":200, "Message":"OK", "Data":{"RiskLevel":"none"}}`
		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(respBody)),
			}, nil
		})

		result, err := m.executeModerationService(context.Background(), "service", "content", nil)
		assert.NoError(t, err)
		assert.True(t, result.Allow)
	})

	t.Run("should fail when api returns business error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		respBody := `{"Code":400, "Message":"Invalid Parameter"}`
		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(respBody)),
			}, nil
		})

		_, err := m.executeModerationService(context.Background(), "service", "content", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aliyun API returned a business error: code=400, message=Invalid Parameter")
	})

	t.Run("should fail when http response is invalid json", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		respBody := `this is not a valid json`
		patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(respBody)),
			}, nil
		})

		_, err := m.executeModerationService(context.Background(), "service", "content", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse Aliyun API response")
	})
}

func TestModerator_RequestAndResponse(t *testing.T) {
	var calledService string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledService = r.URL.Query().Get("Service")
		assert.NotEmpty(t, calledService)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		mockResp := aliResp{
			Code:    200,
			Message: "OK",
			Data: struct {
				Advice []struct {
					HitLabel string `json:"HitLabel"`
					Answer   string `json:"Answer"`
				} `json:"Advice"`
				RiskLevel string `json:"RiskLevel"`
			}{
				RiskLevel: "none",
			},
		}
		respBytes, _ := json.Marshal(mockResp)
		_, _ = w.Write(respBytes)
	}))
	defer mockServer.Close()

	conf := &aicontentsecurity.AliyunConfig{AccessKeyId: "test_id", AccessKeySecret: "test_secret"}
	mod, err := New(&aicontentsecurity.Config_AliyunConfig{AliyunConfig: conf})
	assert.NoError(t, err)

	m, ok := mod.(*Moderator)
	assert.True(t, ok, "m should be of type *Moderator")
	m.endpoint = mockServer.URL

	t.Run("Request should call with llm_query_moderation", func(t *testing.T) {
		calledService = ""
		res, err := m.Request(context.Background(), "hello", nil)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "llm_query_moderation", calledService)
		assert.True(t, res.Allow)
	})

	t.Run("Response should call with llm_response_moderation", func(t *testing.T) {
		calledService = ""
		res, err := m.Response(context.Background(), "world", nil)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "llm_response_moderation", calledService)
		assert.True(t, res.Allow)
	})
}

// TestEvaluateResponse validates the evaluation logic for the API response (updated for the new RiskLevel)
func TestEvaluateResponse(t *testing.T) {
	conf := &aicontentsecurity.AliyunConfig{
		MaxRiskLevel: "medium",
	}
	mod, err := New(&aicontentsecurity.Config_AliyunConfig{AliyunConfig: conf})
	assert.NoError(t, err)
	m, ok := mod.(*Moderator)
	assert.True(t, ok, "m should be of type *Moderator")

	t.Run("should allow when risk level is lower than max", func(t *testing.T) {
		resp := aliResp{}
		_ = json.Unmarshal([]byte(`{"Data":{"RiskLevel":"low"}}`), &resp)
		result, err := m.EvaluateResponse(resp)
		assert.NoError(t, err)
		assert.True(t, result.Allow)
	})

	t.Run("should reject when risk level is equal to max", func(t *testing.T) {
		resp := aliResp{}
		_ = json.Unmarshal([]byte(`{"Data":{"RiskLevel":"medium", "Advice": [{"Answer":"potential risk"}]}}`), &resp)
		result, err := m.EvaluateResponse(resp)
		assert.NoError(t, err)
		assert.False(t, result.Allow)
		assert.Equal(t, "potential risk", result.Reason)
	})

	t.Run("should reject when risk level is higher than max", func(t *testing.T) {
		resp := aliResp{}
		_ = json.Unmarshal([]byte(`{"Data":{"RiskLevel":"high", "Advice": [{"Answer":"high-risk content"}]}}`), &resp)
		result, err := m.EvaluateResponse(resp)
		assert.NoError(t, err)
		assert.False(t, result.Allow)
		assert.Equal(t, "high-risk content", result.Reason)
	})

	t.Run("should allow 'none' risk level", func(t *testing.T) {
		resp := aliResp{}
		_ = json.Unmarshal([]byte(`{"Data":{"RiskLevel":"none"}}`), &resp)
		result, err := m.EvaluateResponse(resp)
		assert.NoError(t, err)
		assert.True(t, result.Allow)
	})

	t.Run("should fail when risk level is invalid", func(t *testing.T) {
		resp := aliResp{}
		_ = json.Unmarshal([]byte(`{"Data":{"RiskLevel":"unknown"}}`), &resp)
		result, err := m.EvaluateResponse(resp)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid risk level: \"unknown\"")
		assert.False(t, result.Allow, "should default to reject when parsing risk level fails")
	})
}

func TestNewModerator(t *testing.T) {
	testCases := []struct {
		name        string
		config      interface{}
		expectError bool
		errorMsg    string
		checkFunc   func(t *testing.T, mod moderation.Moderator)
	}{
		{
			name:        "should return error for invalid config type",
			config:      "not a valid config",
			expectError: true,
			errorMsg:    "invalid config type for aliyun moderator",
		},
		{
			name:        "should return error for nil aliyun config",
			config:      &aicontentsecurity.Config_AliyunConfig{AliyunConfig: nil},
			expectError: true,
			errorMsg:    "aliyun config is empty inside the wrapper",
		},
		{
			name: "should use default values when config is minimal",
			config: &aicontentsecurity.Config_AliyunConfig{
				AliyunConfig: &aicontentsecurity.AliyunConfig{
					AccessKeyId:     "test_id",
					AccessKeySecret: "test_secret",
				},
			},
			expectError: false,
			checkFunc: func(t *testing.T, mod moderation.Moderator) {
				m, ok := mod.(*Moderator)
				require.True(t, ok)
				assert.Equal(t, "https://green-cip.cn-shanghai.aliyuncs.com", m.endpoint)
				assert.Equal(t, 2*time.Second, m.httpClient.Timeout)
				assert.Equal(t, High, m.maxRiskLevel)
				assert.Equal(t, defaultVersion, m.version)
			},
		},
		{
			name: "should override default values from config",
			config: &aicontentsecurity.Config_AliyunConfig{
				AliyunConfig: &aicontentsecurity.AliyunConfig{
					AccessKeyId:     "test_id",
					AccessKeySecret: "test_secret",
					Region:          "cn-beijing",
					Timeout:         "5s",
					MaxRiskLevel:    "low",
					Version:         "2023-01-01",
				},
			},
			expectError: false,
			checkFunc: func(t *testing.T, mod moderation.Moderator) {
				m, ok := mod.(*Moderator)
				require.True(t, ok)
				assert.Equal(t, "https://green-cip.cn-beijing.aliyuncs.com", m.endpoint)
				assert.Equal(t, 5*time.Second, m.httpClient.Timeout)
				assert.Equal(t, Low, m.maxRiskLevel)
				assert.Equal(t, "2023-01-01", m.version)
			},
		},
		{
			name: "should return error for invalid MaxRiskLevel",
			config: &aicontentsecurity.Config_AliyunConfig{
				AliyunConfig: &aicontentsecurity.AliyunConfig{
					MaxRiskLevel: "critical",
				},
			},
			expectError: true,
			errorMsg:    `invalid risk level: "critical"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mod, err := New(tc.config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, mod)
				if tc.checkFunc != nil {
					tc.checkFunc(t, mod)
				}
			}
		})
	}
}

func TestExecuteModerationService_WithSessionId(t *testing.T) {
	var capturedServiceParams string

	testCases := []struct {
		name              string
		useSessionID      bool
		idMap             map[string]string
		expectedSessionID string
		expectSessionID   bool
	}{
		{
			name:              "should include SessionId when enabled and provided",
			useSessionID:      true,
			idMap:             map[string]string{"SessionId": "session-123"},
			expectedSessionID: "session-123",
			expectSessionID:   true,
		},
		{
			name:            "should not include SessionId when enabled but not provided",
			useSessionID:    true,
			idMap:           map[string]string{},
			expectSessionID: false,
		},
		{
			name:            "should not include SessionId when disabled even if provided",
			useSessionID:    false,
			idMap:           map[string]string{"SessionId": "session-123"},
			expectSessionID: false,
		},
		{
			name:            "should not include SessionId when enabled and idMap is nil",
			useSessionID:    true,
			idMap:           nil,
			expectSessionID: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			capturedServiceParams = ""

			confWrapper := &aicontentsecurity.Config_AliyunConfig{
				AliyunConfig: &aicontentsecurity.AliyunConfig{
					AccessKeyId:     "test-id",
					AccessKeySecret: "test-secret",
					UseSessionId:    tc.useSessionID,
				},
			}
			mod, err := New(confWrapper)
			require.NoError(t, err)
			m, ok := mod.(*Moderator)
			require.True(t, ok, "moderator should be of type *Moderator")

			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyMethodFunc(m.httpClient, "Do", func(req *http.Request) (*http.Response, error) {
				queryParams, _ := url.ParseQuery(req.URL.RawQuery)
				capturedServiceParams = queryParams.Get("ServiceParameters")

				responseBody := `{"Code":200, "Message":"OK", "Data":{"RiskLevel":"none"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseBody)),
				}, nil
			})

			_, err = m.executeModerationService(context.Background(), "service", "content", tc.idMap)
			require.NoError(t, err)

			require.NotEmpty(t, capturedServiceParams, "ServiceParameters should have been captured")

			var params map[string]interface{}
			err = json.Unmarshal([]byte(capturedServiceParams), &params)
			require.NoError(t, err, "service parameters should be valid json")

			_, ok = params["SessionId"]
			assert.Equal(t, tc.expectSessionID, ok, "presence of SessionId field should match expectation")
			if tc.expectSessionID {
				assert.Equal(t, tc.expectedSessionID, params["SessionId"])
			}
		})
	}
}

func TestRiskLevel(t *testing.T) {
	t.Run("String representation", func(t *testing.T) {
		assert.Equal(t, "none", None.String())
		assert.Equal(t, "low", Low.String())
		assert.Equal(t, "medium", Medium.String())
		assert.Equal(t, "high", High.String())
		assert.Equal(t, "unknown", RiskLevel(99).String())
	})

	t.Run("ParseRiskLevel", func(t *testing.T) {
		level, err := ParseRiskLevel("medium")
		assert.NoError(t, err)
		assert.Equal(t, Medium, level)

		_, err = ParseRiskLevel("invalid")
		assert.Error(t, err)
		assert.Equal(t, `invalid risk level: "invalid"`, err.Error())
	})

	t.Run("JSON Marshaling", func(t *testing.T) {
		level := Medium
		data, err := json.Marshal(level)
		assert.NoError(t, err)
		assert.Equal(t, `"medium"`, string(data))

		type sampleStruct struct {
			Level RiskLevel `json:"level"`
		}
		s := sampleStruct{Level: High}
		data, err = json.Marshal(s)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"level": "high"}`, string(data))
	})

	t.Run("JSON Unmarshaling", func(t *testing.T) {
		var level RiskLevel
		err := json.Unmarshal([]byte(`"low"`), &level)
		assert.NoError(t, err)
		assert.Equal(t, Low, level)

		err = json.Unmarshal([]byte(`null`), &level)
		assert.NoError(t, err)
		assert.Equal(t, None, level)

		err = json.Unmarshal([]byte(`"critical"`), &level)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `invalid risk level: "critical"`)
	})
}
