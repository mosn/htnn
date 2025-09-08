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

package moderation

import (
	"fmt"
)

type ModeratorFactory func(config interface{}) (Moderator, error)

var registry = make(map[string]ModeratorFactory)

func Register(name string, factory ModeratorFactory) {
	if _, ok := registry[name]; ok {
		panic(fmt.Sprintf("moderator factory named %s already registered", name))
	}
	registry[name] = factory
}

func NewModerator(name string, config interface{}) (Moderator, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no moderator factory registered for name: %s", name)
	}
	return factory(config)
}
