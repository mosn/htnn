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

import "mosn.io/htnn/api/pkg/filtermanager/api"

type Extractor interface {

	// SetData parse the raw data and prepare the internal state for subsequent extraction calls.
	SetData(data []byte) error

	// RequestContent extracts the request content from the data loaded previously.
	RequestContent() string
	// ResponseContent extracts the response content from the data loaded previously.
	ResponseContent() string
	// StreamResponseContent extracts the stream response content from the data loaded previously.
	StreamResponseContent() string

	// IDsFromRequestData extracts IDs from the loaded data body and populates the given idMap.
	IDsFromRequestData(idMap map[string]string)

	// IDsFromRequestHeaders extracts IDs from the given headers and populates the idMap.
	IDsFromRequestHeaders(headers api.RequestHeaderMap, idMap map[string]string)
}
