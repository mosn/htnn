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

package extractor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/plugins/plugins/limitToken/extractor"
	"mosn.io/htnn/types/plugins/limitToken"
)

func buildGjsonConfig() *limitToken.Config_GjsonConfig {
	return &limitToken.Config_GjsonConfig{
		GjsonConfig: &limitToken.GjsonConfig{
			RequestContentPath:           "request.content",
			RequestModelPath:             "request.model",
			ResponseContentPath:          "response.content",
			ResponseModelPath:            "response.model",
			ResponseCompletionTokensPath: "response.usage.completion_tokens",
			ResponsePromptTokensPath:     "response.usage.prompt_tokens",
			StreamResponseContentPath:    "stream.content",
			StreamResponseModelPath:      "stream.model",
		},
	}
}

func TestGjsonExtractor_RequestContentAndModel(t *testing.T) {
	cfg := buildGjsonConfig()
	ex, err := extractor.New(cfg)
	require.NoError(t, err)

	// 正常 JSON
	data := []byte(`{
		"request": {"content": "hello", "model": "gpt-4"},
		"response": {"content": "world", "model": "gpt-4o", "usage": {"completion_tokens": 12, "prompt_tokens": 8}},
		"stream": {"content": "chunk", "model": "gpt-4s"}
	}`)
	err = ex.SetData(data)
	require.NoError(t, err)

	content, model := ex.RequestContentAndModel()
	assert.Equal(t, "hello", content)
	assert.Equal(t, "gpt-4", model)
}

func TestGjsonExtractor_ResponseContentAndModel(t *testing.T) {
	cfg := buildGjsonConfig()
	ex, _ := extractor.New(cfg)

	data := []byte(`{
		"response": {"content": "world", "model": "gpt-4o", "usage": {"completion_tokens": 12, "prompt_tokens": 8}}
	}`)
	err := ex.SetData(data)
	require.NoError(t, err)

	content, model, completionTokens, promptTokens := ex.ResponseContentAndModel()
	assert.Equal(t, "world", content)
	assert.Equal(t, "gpt-4o", model)
	assert.Equal(t, int64(12), completionTokens)
	assert.Equal(t, int64(8), promptTokens)
}

func TestGjsonExtractor_StreamResponseContentAndModel(t *testing.T) {
	cfg := buildGjsonConfig()
	ex, _ := extractor.New(cfg)

	data := []byte(`{"stream": {"content": "chunk", "model": "gpt-4s"}}`)
	err := ex.SetData(data)
	require.NoError(t, err)

	content, model := ex.StreamResponseContentAndModel()
	assert.Equal(t, "chunk", content)
	assert.Equal(t, "gpt-4s", model)
}

func TestGjsonExtractor_InvalidJSON(t *testing.T) {
	cfg := buildGjsonConfig()
	ex, _ := extractor.New(cfg)

	data := []byte(`invalid json`)
	err := ex.SetData(data)
	require.Error(t, err)

	content, model := ex.RequestContentAndModel()
	assert.Equal(t, "", content)
	assert.Equal(t, "", model)

	content, model, cTokens, pTokens := ex.ResponseContentAndModel()
	assert.Equal(t, "", content)
	assert.Equal(t, "", model)
	assert.Equal(t, int64(0), cTokens)
	assert.Equal(t, int64(0), pTokens)

	content, model = ex.StreamResponseContentAndModel()
	assert.Equal(t, "", content)
	assert.Equal(t, "", model)
}

func TestGjsonExtractor_MissingConfig(t *testing.T) {
	ex, err := extractor.New(&limitToken.Config_GjsonConfig{
		GjsonConfig: &limitToken.GjsonConfig{},
	})
	require.NoError(t, err)

	data := []byte(`{"request": {"content":"hi"}}`)
	_ = ex.SetData(data)

	content, model := ex.RequestContentAndModel()
	assert.Equal(t, "", content)
	assert.Equal(t, "", model)
}
