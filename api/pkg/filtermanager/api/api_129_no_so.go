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

//go:build envoy-1.29 && !so

package api

import (
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

// SetLogLevel is used only for test! It doesn't exist when we build as so.
func SetLogLevel(level api.LogType) {
	currLogLevel.Store(int32(level))
}
