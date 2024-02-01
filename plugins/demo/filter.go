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

package demo

import (
	"fmt"

	"mosn.io/htnn/pkg/filtermanager/api"
)

// configFactory returns a factory that produces per-request Filter.
// You can use it to bind the configuration and do per-request initialization.
func configFactory(c interface{}) api.FilterFactory {
	conf := c.(*config)
	return func(callbacks api.FilterCallbackHandler) api.Filter {
		return &filter{
			callbacks: callbacks,
			config:    conf,
		}
	}
}

type filter struct {
	// PassThroughFilter is the base class of filter which provides the default implementation
	// to Filter methods - do nothing.
	api.PassThroughFilter

	// callbacks provides the API we can use to implement filter's feature
	callbacks api.FilterCallbackHandler
	config    *config
}

// The doc of each API can be found in package pkg/filtermanager/api

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	headers.Add(f.config.HostName, f.hello())
	return api.Continue
}

func (f *filter) hello() string {
	name := f.callbacks.StreamInfo().FilterState().GetString("guest_name")
	api.LogInfo("hello")
	return fmt.Sprintf("hello, %s", name)
}
