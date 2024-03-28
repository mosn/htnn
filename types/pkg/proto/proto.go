// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package proto

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// UnmarshalJSON should be used instead of protojson.Unmarshal. The first one contains custom options.
func UnmarshalJSON(data []byte, conf proto.Message) error {
	// Don't throw an error when there is unknown field. Therefore, we can rollback to previous
	// version quickly under the same configurations, and make A/B test easier.
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(data, conf)
}

// UnmarshalJSONStrictly works like UnmarshalJSON, but returns error when the data contains unknown field.
func UnmarshalJSONStrictly(data []byte, conf proto.Message) error {
	return protojson.UnmarshalOptions{}.Unmarshal(data, conf)
}
