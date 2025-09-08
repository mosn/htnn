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

package localservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestLocalService_moderateContent(t *testing.T) {
	s := &LocalService{
		client:             &http.Client{},
		unhealthyWords:     []string{"unhealthy"},
		customErrorMessage: "custom error",
	}

	testCases := []struct {
		name                   string
		inputContent           string
		mockHTTPStatus         int
		mockHTTPBody           string
		mockHTTPError          error
		expectedAllow          bool
		expectedReasonContains string
		expectedErrorContains  string
	}{
		{
			name:           "should allow safe content",
			inputContent:   "this is a good sentence",
			mockHTTPStatus: http.StatusOK,
			mockHTTPBody:   `{"is_safe": true}`,
			expectedAllow:  true,
		},
		{
			name:                   "should deny with a reason",
			inputContent:           "a bad sentence",
			mockHTTPStatus:         http.StatusOK,
			mockHTTPBody:           `{"is_safe": false, "flagged_words": ["bad"]}`,
			expectedAllow:          false,
			expectedReasonContains: "contains inappropriate words: bad",
		},
		{
			name:                   "should deny with default message if no reason provided",
			inputContent:           "another bad sentence",
			mockHTTPStatus:         http.StatusOK,
			mockHTTPBody:           `{"is_safe": false}`,
			expectedAllow:          false,
			expectedReasonContains: "content flagged as inappropriate",
		},
		{
			name:                  "should return error when http client fails",
			inputContent:          "any content",
			mockHTTPStatus:        0,
			mockHTTPError:         errors.New("network timeout"),
			expectedErrorContains: "failed to send request: network timeout",
		},
		{
			name:                  "should return error on non-200 status code",
			inputContent:          "any content",
			mockHTTPStatus:        http.StatusInternalServerError,
			mockHTTPBody:          "server is down",
			expectedErrorContains: "moderation service returned status 500: server is down",
		},
		{
			name:                  "should return error on malformed json response",
			inputContent:          "any content",
			mockHTTPStatus:        http.StatusOK,
			mockHTTPBody:          `{"is_safe": true,}`,
			expectedErrorContains: "failed to decode response",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			var capturedRequest moderationRequest

			patches.ApplyMethodFunc(s.client, "Do", func(req *http.Request) (*http.Response, error) {
				bodyBytes, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(bodyBytes, &capturedRequest)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if tc.mockHTTPError != nil {
					return nil, tc.mockHTTPError
				}
				if tc.mockHTTPStatus == 0 {
					return nil, nil
				}

				return mockResponse(tc.mockHTTPStatus, tc.mockHTTPBody), nil
			})

			result, err := s.moderateContent(context.Background(), tc.inputContent)

			if tc.expectedErrorContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedAllow, result.Allow)
				if tc.expectedReasonContains != "" {
					assert.Contains(t, result.Reason, tc.expectedReasonContains)
				}

				assert.Equal(t, tc.inputContent, capturedRequest.Content)
				assert.Equal(t, s.unhealthyWords, capturedRequest.UnhealthyWords)
				assert.Equal(t, s.customErrorMessage, capturedRequest.CustomErrorMessage)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("should create LocalService successfully with valid config", func(t *testing.T) {
		conf := &aicontentsecurity.Config_LocalModerationServiceConfig{
			LocalModerationServiceConfig: &aicontentsecurity.LocalModerationServiceConfig{
				BaseUrl: "http://test.com",
				Timeout: "5s",
			},
		}
		m, err := New(conf)
		require.NoError(t, err)
		require.NotNil(t, m)

		s, ok := m.(*LocalService)
		require.True(t, ok)
		assert.Equal(t, "http://test.com", s.baseURL)
		assert.Equal(t, 5*time.Second, s.client.Timeout)
	})

	t.Run("should return error for invalid config type", func(t *testing.T) {
		_, err := New(&aicontentsecurity.Config_AliyunConfig{})
		assert.Error(t, err)
	})

	t.Run("should return error for nil config", func(t *testing.T) {
		_, err := New(&aicontentsecurity.Config_LocalModerationServiceConfig{
			LocalModerationServiceConfig: nil,
		})
		assert.Error(t, err)
	})
}
