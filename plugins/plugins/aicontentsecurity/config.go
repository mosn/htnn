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

package aicontentsecurity

import (
	"reflect"
	"time"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/pkg/plugins"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/extractor"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func init() {
	plugins.RegisterPlugin(aicontentsecurity.Name, &plugin{})
}

type plugin struct {
	aicontentsecurity.Plugin
}

func (p *plugin) Factory() api.FilterFactory {
	return factory
}

func (p *plugin) Config() api.PluginConfig {
	return &config{}
}

type config struct {
	aicontentsecurity.CustomConfig

	moderator         moderation.Moderator
	extractor         extractor.Extractor
	moderationTimeout time.Duration
}

func (conf *config) Init(cb api.ConfigCallbackHandler) error {
	if conf.ModerationTimeout != "" {
		conf.moderationTimeout, _ = time.ParseDuration(conf.ModerationTimeout)
	} else {
		conf.moderationTimeout = 3 * time.Second
	}

	providerTypeName := reflect.TypeOf(conf.ProviderConfig).String()
	moderator, err := moderation.NewModerator(providerTypeName, conf.ProviderConfig)
	if err != nil {
		api.LogErrorf("failed to create moderator for provider type '%s': %v", providerTypeName, err)
		return err
	}
	conf.moderator = moderator

	extractorTypeName := reflect.TypeOf(conf.ExtractorConfig).String()
	newExtractor, err := extractor.NewExtractor(extractorTypeName, conf.ExtractorConfig)
	if err != nil {
		api.LogErrorf("failed to create newExtractor for provider type '%s': %v", extractorTypeName, err)
		return err
	}
	conf.extractor = newExtractor

	return nil
}
