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

type Extractor interface {

	// SetData parse the raw data and prepare the internal state for subsequent extraction calls.
	SetData(data []byte) error

	// RequestContentAndModel extracts the request content from the data loaded previously.
	RequestContentAndModel() (string, string)
	// ResponseContentAndModel extracts the response content from the data loaded previously.
	ResponseContentAndModel() (string, string, int64, int64)
	// StreamResponseContentAndModel extracts the stream response content from the data loaded previously.
	StreamResponseContentAndModel() (string, string)
}
