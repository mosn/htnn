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

package integration

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/extractor"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/sseparser"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

var (
	//go:embed testdata/ai_mock_service_route.yml
	aiContentSecurityRoute string

	//go:embed testdata/ai_mock_service_cluster.yml
	aiContentSecurityCluster string
)

func TestAIContentSecurity(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		Bootstrap: dataplane.Bootstrap().AddBackendRoute(aiContentSecurityRoute).AddCluster(aiContentSecurityCluster),
	})

	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":10901", "llm-mock")
	helper.WaitServiceUp(t, ":10902", "llm-moderation")

	customModerationErrorMsg := "The content you sent includes inappropriate information and has been intercepted by the system."

	config := controlplane.NewSinglePluginConfig("AIContentSecurity", map[string]interface{}{
		"moderation_timeout":              "3000ms",
		"streaming_enabled":               true,
		"moderation_char_limit":           5,
		"moderation_chunk_overlap_length": 3,
		"local_moderation_service_config": map[string]interface{}{
			"base_url":             "http://aimockservices:10902",
			"unhealthy_words":      []string{"hate", "ugly"},
			"custom_error_message": customModerationErrorMsg,
		},
		"gjson_config": map[string]interface{}{
			"request_content_path":         "content",
			"response_content_path":        "content",
			"stream_response_content_path": "choices.0.delta.content",
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", dp.Port())

	t.Run("Non-streaming sanity", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"response_message": "This is a normal non-streaming message.",
			"content":          "dasdasdads",
			"stream":           false,
		}
		jsonBody, err := json.Marshal(requestBody)
		assert.NoError(t, err, "failed to marshal request body")

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		assert.NoError(t, err, "failed to create request")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err, "request failed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected status code 200")

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err, "failed to read response body")

		var result map[string]interface{}
		assert.NoError(t, json.Unmarshal(body, &result), "failed to unmarshal response body")
		assert.Contains(t, result, "content", "expected 'content' key in response")
	})

	t.Run("Non-streaming error", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"response_message": "this message contains hate and should be blocked",
			"stream":           false,
			"content":          "dasdasdads",
		}
		jsonBody, err := json.Marshal(requestBody)
		assert.NoError(t, err, "failed to marshal request body")

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		assert.NoError(t, err, "failed to create request")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		assert.NoError(t, err, "request failed")
		defer resp.Body.Close()

		assert.NotEqual(t, http.StatusOK, resp.StatusCode, "expected a non-200 status code for a blocked request")

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err, "failed to read response body")

		assert.Contains(t, string(body), customModerationErrorMsg, "unexpected error message")
	})

	t.Run("Streaming sanity", func(t *testing.T) {
		expectedMinContentEvents := 4
		content := "asd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads a"
		requestBody := map[string]interface{}{
			"response_message": content,
			"content":          "dasdasdads",
			"stream":           true,
			"event_num":        expectedMinContentEvents,
		}
		jsonBody, err := json.Marshal(requestBody)
		assert.NoError(t, err, "failed to marshal request body")

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		assert.NoError(t, err, "failed to create request")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		assert.NoError(t, err, "request failed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected status code 200")

		parser := sseparser.NewStreamEventParser()
		extractor, err := extractor.New(&aicontentsecurity.Config_GjsonConfig{
			GjsonConfig: &aicontentsecurity.GjsonConfig{
				StreamResponseContentPath: "choices.0.delta.content",
			},
		})
		assert.NoError(t, err, "failed to create extractor")

		var finalContent strings.Builder
		var eventCount int
		buf := make([]byte, 1024)

		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				parser.Append(buf[:n])

				for {
					event, parseErr := parser.TryParse()
					assert.NoError(t, parseErr, "SSE data format error")
					if event == nil {
						break
					}

					_ = extractor.SetData([]byte(event.Data))
					content := extractor.StreamResponseContent()
					finalContent.WriteString(content)
					if content != "" {
						eventCount++
						fmt.Printf("Parsed event %d, content: '%s'\n", eventCount, content)
					}
					parser.Consume(1)
				}
			}

			if err != nil {
				if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
					fmt.Println("Stream finished.")
					break
				}
				t.Fatalf("error reading stream: %v", err)
			}
		}

		responseStr := finalContent.String()
		fmt.Printf("\nTotal content events: %d\n", eventCount)
		fmt.Printf("Final assembled content: %s\n", responseStr)

		assert.True(t, eventCount == expectedMinContentEvents, "should receive at least %d content events, but got %d", expectedMinContentEvents, eventCount)
		assert.Equal(t, content, responseStr, "final content should have the correct prefix")
	})

	t.Run("Streaming error", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"response_message": "hate,this streaming message contains hate and should be blocked",
			"content":          "dasdasdads",
			"stream":           true,
		}
		jsonBody, err := json.Marshal(requestBody)
		assert.NoError(t, err, "failed to marshal request body")

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		assert.NoError(t, err, "failed to create request")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		assert.NoError(t, err, "request failed")
		defer resp.Body.Close()

		parser := sseparser.NewStreamEventParser()
		buf := make([]byte, 1024)

		var errorEventFound bool
		var errorMessage string

		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				parser.Append(buf[:n])

				for {
					event, parseErr := parser.TryParse()
					if parseErr != nil && !errorEventFound {
						assert.NoError(t, parseErr, "SSE data format error")
					}
					if event == nil {
						break
					}

					if event.Event == "error" {
						errorEventFound = true
						errorMessage = event.Data
					}

					if errorEventFound {
						break
					}
				}
			}

			if errorEventFound {
				break
			}

			if err != nil {
				if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) {
					fmt.Println("Stream finished.")
					break
				}
				t.Fatalf("error reading stream: %v", err)
			}
		}

		assert.True(t, errorEventFound, "expected to find an 'error' event in the stream but did not")
		assert.Contains(t, errorMessage, customModerationErrorMsg, "the error event data should contain the custom error message")
	})

	config = controlplane.NewSinglePluginConfig("AIContentSecurity", map[string]interface{}{
		"moderation_timeout":              "3000ms",
		"streaming_enabled":               false,
		"moderation_char_limit":           5,
		"moderation_chunk_overlap_length": 3,
		"local_moderation_service_config": map[string]interface{}{
			"base_url":             "http://aimockservices:10902",
			"unhealthy_words":      []string{"hate", "ugly"},
			"custom_error_message": customModerationErrorMsg,
		},
		"gjson_config": map[string]interface{}{
			"request_content_path":         "content",
			"response_content_path":        "content",
			"stream_response_content_path": "choices.0.delta.content",
		},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	t.Run("Streaming sanity - check full", func(t *testing.T) {
		expectedMinContentEvents := 4
		content := "asd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads aasd asda sdasd ads a"
		requestBody := map[string]interface{}{
			"response_message": content,
			"content":          "dasdasdads",
			"stream":           true,
			"event_num":        expectedMinContentEvents,
		}
		jsonBody, err := json.Marshal(requestBody)
		assert.NoError(t, err, "failed to marshal request body")

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		assert.NoError(t, err, "failed to create request")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		assert.NoError(t, err, "request failed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "expected status code 200")

		allData, err := io.ReadAll(resp.Body)
		assert.NoError(t, err, "failed to read full body")

		parser := sseparser.NewStreamEventParser()
		parser.Append(allData)

		extractor, err := extractor.New(&aicontentsecurity.Config_GjsonConfig{
			GjsonConfig: &aicontentsecurity.GjsonConfig{
				StreamResponseContentPath: "choices.0.delta.content",
			},
		})
		assert.NoError(t, err, "failed to create extractor")

		var finalContent strings.Builder
		var eventCount int

		for {
			event, parseErr := parser.TryParse()
			assert.NoError(t, parseErr, "SSE data format error")
			if event == nil {
				break
			}

			_ = extractor.SetData([]byte(event.Data))
			seg := extractor.StreamResponseContent()
			finalContent.WriteString(seg)
			if seg != "" {
				eventCount++
				fmt.Printf("Parsed event %d, content: '%s'\n", eventCount, seg)
			}
			parser.Consume(1)
		}

		responseStr := finalContent.String()
		fmt.Printf("\nTotal content events: %d\n", eventCount)
		fmt.Printf("Final assembled content: %s\n", responseStr)

		assert.Equal(t, expectedMinContentEvents, eventCount,
			"should receive exactly %d content events, but got %d", expectedMinContentEvents, eventCount)
		assert.Equal(t, content, responseStr, "final content should match expected message")
	})

}
