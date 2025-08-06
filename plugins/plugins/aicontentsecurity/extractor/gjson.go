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

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func init() {
	var cfg *aicontentsecurity.Config_GjsonConfig
	typeName := reflect.TypeOf(cfg).String()
	Register(typeName, New)
}

type GjsonContentExtractor struct {
	data       []byte
	config     *aicontentsecurity.GjsonConfig
	parsedData gjson.Result
}

func New(config interface{}) (Extractor, error) {
	wrapper, ok := config.(*aicontentsecurity.Config_GjsonConfig)
	if !ok {
		return nil, errors.New("invalid config type for GjsonContentExtractor")
	}

	configWrapper := wrapper.GjsonConfig
	if configWrapper == nil {
		return nil, errors.New("GjsonContentExtractor config is empty inside the wrapper")
	}

	return &GjsonContentExtractor{
		config: configWrapper,
	}, nil
}

func (g *GjsonContentExtractor) SetData(data []byte) error {
	g.data = nil
	g.parsedData = gjson.Result{}
	if len(data) == 0 || !gjson.ValidBytes(data) {
		return errors.New("invalid json data")
	} else {
		g.parsedData = gjson.ParseBytes(data)
		return nil
	}
}

func (g *GjsonContentExtractor) RequestContent() string {
	if !g.parsedData.Exists() || g.config == nil || g.config.RequestContentPath == "" {
		return ""
	}

	result := g.parsedData.Get(g.config.RequestContentPath)
	if !result.Exists() {
		return ""
	}

	return result.String()
}

func (g *GjsonContentExtractor) ResponseContent() string {
	if !g.parsedData.Exists() || g.config == nil || g.config.ResponseContentPath == "" {
		return ""
	}

	result := g.parsedData.Get(g.config.ResponseContentPath)
	if !result.Exists() {
		return ""
	}

	return result.String()
}

func (g *GjsonContentExtractor) StreamResponseContent() string {
	if !g.parsedData.Exists() || g.config == nil || g.config.StreamResponseContentPath == "" {
		return ""
	}

	result := g.parsedData.Get(g.config.StreamResponseContentPath)
	if !result.Exists() {
		return ""
	}

	return result.String()
}

func (g *GjsonContentExtractor) IDsFromRequestHeaders(headers api.RequestHeaderMap, idMap map[string]string) {
	if g.config == nil || g.config.HeaderFields == nil {
		return
	}

	for _, field := range g.config.HeaderFields {
		if field.SourceField == "" || field.TargetField == "" {
			continue
		}

		headerValue, exists := headers.Get(field.SourceField)
		if exists && headerValue != "" {
			idMap[field.TargetField] = headerValue
		}
	}
}

func (g *GjsonContentExtractor) IDsFromRequestData(idMap map[string]string) {
	if !g.parsedData.Exists() || g.config == nil || g.config.BodyFields == nil {
		return
	}

	for _, field := range g.config.BodyFields {
		if field.GetSourceField() == "" || field.GetTargetField() == "" {
			continue
		}

		result := g.parsedData.Get(field.GetSourceField())
		if result.Exists() {
			idMap[field.TargetField] = result.String()
		}
	}
}
