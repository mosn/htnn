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

//nolint:gosec
import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func init() {
	var cfg *aicontentsecurity.Config_AliyunConfig
	typeName := reflect.TypeOf(cfg).String()
	moderation.Register(typeName, New)
}

const (
	defaultFormat           = "JSON"
	defaultSignatureMethod  = "HMAC-SHA1"
	defaultSignatureVersion = "1.0"
	defaultAction           = "TextModerationPlus"
	defaultVersion          = "2022-03-02"
	defaultEndpoint         = "https://green-cip.cn-shanghai.aliyuncs.com"
	endpointTemplate        = "https://green-cip.%s.aliyuncs.com"
)

func percentEncode(str string) string {
	encoded := url.QueryEscape(str)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

func generateNonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

type Moderator struct {
	accessKeyID      string
	accessKeySecret  string
	endpoint         string
	format           string
	version          string
	signatureMethod  string
	signatureVersion string
	action           string

	httpClient *http.Client
	config     *aicontentsecurity.AliyunConfig

	maxRiskLevel RiskLevel
}

type aliResp struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
	Data    struct {
		Advice []struct {
			HitLabel string `json:"HitLabel"`
			Answer   string `json:"Answer"`
		} `json:"Advice"`
		RiskLevel string `json:"RiskLevel"`
	} `json:"Data"`
}

func New(config interface{}) (moderation.Moderator, error) {
	wrapper, ok := config.(*aicontentsecurity.Config_AliyunConfig)
	if !ok {
		return nil, errors.New("invalid config type for aliyun moderator")
	}

	conf := wrapper.AliyunConfig
	if conf == nil {
		return nil, errors.New("aliyun config is empty inside the wrapper")
	}

	m := &Moderator{
		accessKeyID:      conf.GetAccessKeyId(),
		accessKeySecret:  conf.GetAccessKeySecret(),
		format:           defaultFormat,
		signatureMethod:  defaultSignatureMethod,
		signatureVersion: defaultSignatureVersion,
		action:           defaultAction,
		version:          defaultVersion,
		endpoint:         defaultEndpoint,
		config:           conf,
		maxRiskLevel:     High,
	}

	if rl := conf.GetMaxRiskLevel(); rl != "" {
		level, err := ParseRiskLevel(rl)
		if err != nil {
			return nil, err
		}
		m.maxRiskLevel = level
	}

	if clientTimeout := conf.GetTimeout(); clientTimeout > 0 {
		m.httpClient = &http.Client{Timeout: time.Duration(clientTimeout) * time.Millisecond}
	} else {
		m.httpClient = &http.Client{Timeout: time.Duration(1) * time.Second}
	}

	if region := conf.GetRegion(); region != "" {
		m.endpoint = fmt.Sprintf(endpointTemplate, region)
	}
	if version := conf.GetVersion(); version != "" {
		m.version = version
	}

	return m, nil
}

func (m *Moderator) generateSignature(params map[string]string) string {
	var keys []string
	for k := range params {
		if k != "Signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var canonicalizedQueryString strings.Builder
	for i, k := range keys {
		if i > 0 {
			canonicalizedQueryString.WriteString("&")
		}
		canonicalizedQueryString.WriteString(percentEncode(k))
		canonicalizedQueryString.WriteString("=")
		canonicalizedQueryString.WriteString(percentEncode(params[k]))
	}

	stringToSign := fmt.Sprintf("POST&%s&%s",
		percentEncode("/"),
		percentEncode(canonicalizedQueryString.String()))

	key := m.accessKeySecret + "&"
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

func (m *Moderator) call(ctx context.Context, service string, serviceParameters string) ([]byte, error) {
	params := map[string]string{
		"Format":            m.format,
		"Version":           m.version,
		"AccessKeyId":       m.accessKeyID,
		"SignatureMethod":   m.signatureMethod,
		"Timestamp":         getTimestamp(),
		"SignatureVersion":  m.signatureVersion,
		"SignatureNonce":    generateNonce(),
		"Action":            m.action,
		"Service":           service,
		"ServiceParameters": serviceParameters,
	}

	signature := m.generateSignature(params)
	params["Signature"] = signature

	queryParams := url.Values{}
	for k, v := range params {
		queryParams.Set(k, v)
	}

	fullURL := fmt.Sprintf("%s?%s", m.endpoint, queryParams.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (m *Moderator) executeModerationService(ctx context.Context, serviceName string, content string, idMap map[string]string) (*moderation.Result, error) {
	serviceParams := map[string]string{
		"content": content,
	}
	if m.config.UseSessionId {
		if sessionID, ok := idMap["SessionId"]; ok && sessionID != "" {
			serviceParams["SessionId"] = sessionID
		}
	}
	serviceParamsJSON, err := json.Marshal(serviceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize service parameters: %w", err)
	}

	rawRespBody, err := m.call(ctx, serviceName, string(serviceParamsJSON))
	if err != nil {
		return nil, err
	}

	var aliAPIResponse aliResp
	if err := json.Unmarshal(rawRespBody, &aliAPIResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Aliyun API response: %w, body: %s", err, string(rawRespBody))
	}

	if aliAPIResponse.Code != 200 {
		return nil, fmt.Errorf("aliyun API returned a business error: code=%d, message=%s", aliAPIResponse.Code, aliAPIResponse.Message)
	}

	return m.EvaluateResponse(aliAPIResponse)
}

func (m *Moderator) Request(ctx context.Context, content string, idMap map[string]string) (*moderation.Result, error) {
	return m.executeModerationService(ctx, "llm_query_moderation", content, idMap)
}

func (m *Moderator) Response(ctx context.Context, content string, idMap map[string]string) (*moderation.Result, error) {
	return m.executeModerationService(ctx, "llm_response_moderation", content, idMap)
}

func (m *Moderator) EvaluateResponse(aliyunResp aliResp) (*moderation.Result, error) {
	evaluationResult := moderation.Result{}
	receivedRiskLevel, err := ParseRiskLevel(aliyunResp.Data.RiskLevel)
	if err != nil {
		evaluationResult.Allow = false
		return &evaluationResult, err
	}

	if receivedRiskLevel >= m.maxRiskLevel {
		evaluationResult.Allow = false
		if len(aliyunResp.Data.Advice) > 0 {
			evaluationResult.Reason = aliyunResp.Data.Advice[0].Answer
		}
		return &evaluationResult, nil
	}

	evaluationResult.Allow = true
	return &evaluationResult, nil
}
