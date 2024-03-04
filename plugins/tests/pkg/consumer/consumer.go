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

package consumer

import (
	"mosn.io/htnn/internal/consumer"
	"mosn.io/htnn/pkg/filtermanager/api"
)

// NewConsumer creates an api.Consumer which can be used to test consumer plugin
func NewConsumer(pluginConsumerConfig map[string]api.PluginConsumerConfig) api.Consumer {
	return &consumer.Consumer{
		ConsumerConfigs: pluginConsumerConfig,
	}
}
