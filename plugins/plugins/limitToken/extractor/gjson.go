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

package extractor

import (
	"errors"
	"reflect"

	"github.com/tidwall/gjson"
	"mosn.io/htnn/types/plugins/limitToken"
)

func init() {
	var cfg *limitToken.Config_GjsonConfig
	typeName := reflect.TypeOf(cfg).String()
	Register(typeName, New)
}

type GjsonExtractor struct {
	data       []byte
	config     *limitToken.GjsonConfig
	parsedData gjson.Result
}

func New(config interface{}) (Extractor, error) {
	wrapper, ok := config.(*limitToken.Config_GjsonConfig)
	if !ok {
		return nil, errors.New("invalid config type for GjsonExtractor")
	}

	configWrapper := wrapper.GjsonConfig
	if configWrapper == nil {
		return nil, errors.New("GjsonExtractor config is empty inside the wrapper")
	}

	return &GjsonExtractor{
		config: configWrapper,
	}, nil
}

func (g *GjsonExtractor) SetData(data []byte) error {
	g.data = nil
	g.parsedData = gjson.Result{}
	if len(data) == 0 || !gjson.ValidBytes(data) {
		return errors.New("invalid json data")
	} else {
		g.parsedData = gjson.ParseBytes(data)
		return nil
	}
}

func (g *GjsonExtractor) RequestContentAndModel() (string, string) {
	if !g.parsedData.Exists() || g.config == nil || g.config.RequestContentPath == "" {
		return "", ""
	}

	content := g.parsedData.Get(g.config.RequestContentPath)
	if !content.Exists() {
		return "", ""
	}
	model := g.parsedData.Get(g.config.RequestModelPath)
	if !model.Exists() {
		return "", ""
	}

	return content.String(), model.String()
}

func (g *GjsonExtractor) ResponseContentAndModel() (string, string, int64, int64) {
	if !g.parsedData.Exists() || g.config == nil || g.config.ResponseContentPath == "" || g.config.ResponseModelPath == "" {
		return "", "", 0, 0
	}

	content := g.parsedData.Get(g.config.ResponseContentPath)
	if !content.Exists() {
		return "", "", 0, 0
	}

	model := g.parsedData.Get(g.config.ResponseModelPath)
	if !model.Exists() {
		return "", "", 0, 0
	}

	completionTokens := g.parsedData.Get(g.config.ResponseCompletionTokensPath)
	promptTokens := g.parsedData.Get(g.config.ResponsePromptTokensPath)
	return content.String(), model.String(), completionTokens.Int(), promptTokens.Int()
}

func (g *GjsonExtractor) StreamResponseContentAndModel() (string, string) {
	if !g.parsedData.Exists() || g.config == nil || g.config.StreamResponseContentPath == "" {
		return "", ""
	}

	content := g.parsedData.Get(g.config.StreamResponseContentPath)
	if !content.Exists() {
		return "", ""
	}

	model := g.parsedData.Get(g.config.StreamResponseModelPath)
	if !model.Exists() {
		return "", ""
	}

	return content.String(), model.String()
}
