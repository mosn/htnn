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

package sentinel

import (
	"mosn.io/htnn/api/pkg/filtermanager/api"
	types "mosn.io/htnn/types/plugins/sentinel"

	sentinel "github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/base"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
}

// Sentinel traffic control
func (f *filter) verify(resName string) bool {
	e, b := sentinel.Entry(resName, sentinel.WithTrafficType(base.Inbound), sentinel.WithArgs())
	if b != nil {
		// blocked
		return false
	}

	// passed
	e.Exit()
	return true
}

// TODO(WeixinX): 完善限流逻辑
func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	config := f.config
	var vals []string
	var resName string
	if config.Key.Source == types.Key_HEADER {
		vals = headers.Values(config.Key.Name)
	} else {
		vals = headers.URL().Query()[config.Key.Name]
	}

	if len(vals) >= 1 {
		resName = vals[0]
	}

	if ok := f.verify(resName); !ok {
		return &api.LocalResponse{Code: 429}
	}
	return api.Continue
}
