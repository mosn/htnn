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
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func init() {
	var cfg *aicontentsecurity.Config_LocalModerationServiceConfig
	typeName := reflect.TypeOf(cfg).String()
	moderation.Register(typeName, New)
}

type LocalService struct {
	client             *http.Client
	baseURL            string
	unhealthyWords     []string
	customErrorMessage string
}

func New(config interface{}) (moderation.Moderator, error) {
	wrapper, ok := config.(*aicontentsecurity.Config_LocalModerationServiceConfig)
	if !ok {
		return nil, errors.New("invalid config type for local moderator")
	}

	conf := wrapper.LocalModerationServiceConfig
	if conf == nil {
		return nil, errors.New("LocalModerationService config is empty inside the wrapper")
	}

	var timeout time.Duration
	if conf.GetTimeout() != "" {
		timeout, _ = time.ParseDuration(conf.GetTimeout())

	}

	return &LocalService{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL:            conf.BaseUrl,
		unhealthyWords:     conf.UnhealthyWords,
		customErrorMessage: conf.CustomErrorMessage,
	}, nil
}

type moderationRequest struct {
	Content            string   `json:"content"`
	UnhealthyWords     []string `json:"unhealthy_words"`
	CustomErrorMessage string   `json:"custom_error_message,omitempty"`
}

type moderationResponse struct {
	IsSafe       bool     `json:"is_safe"`
	FlaggedWords []string `json:"flagged_words"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

func (s *LocalService) Request(ctx context.Context, content string, _ map[string]string) (*moderation.Result, error) {
	return s.moderateContent(ctx, content)
}

func (s *LocalService) Response(ctx context.Context, content string, _ map[string]string) (*moderation.Result, error) {
	return s.moderateContent(ctx, content)
}

func (s *LocalService) moderateContent(ctx context.Context, content string) (*moderation.Result, error) {
	reqPayload := moderationRequest{
		Content:            content,
		UnhealthyWords:     s.unhealthyWords,
		CustomErrorMessage: s.customErrorMessage,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/audit", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("moderation service returned status %d: %s", resp.StatusCode, string(body))
	}

	var mResp moderationResponse
	if err := json.NewDecoder(resp.Body).Decode(&mResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &moderation.Result{
		Allow: mResp.IsSafe,
	}
	if !mResp.IsSafe {
		if mResp.ErrorMessage != "" {
			result.Reason = mResp.ErrorMessage
		} else if len(mResp.FlaggedWords) > 0 {
			result.Reason = fmt.Sprintf("content contains inappropriate words: %s", strings.Join(mResp.FlaggedWords, ", "))
		} else {
			result.Reason = "content flagged as inappropriate by moderation service"
		}
	}

	return result, nil
}
