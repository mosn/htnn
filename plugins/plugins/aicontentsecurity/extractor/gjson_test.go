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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func NewGjsonContentExtractor(config *aicontentsecurity.GjsonConfig) *GjsonContentExtractor {
	return &GjsonContentExtractor{
		config: config,
	}
}

func TestSetData(t *testing.T) {
	extractor := &GjsonContentExtractor{config: &aicontentsecurity.GjsonConfig{}}

	testCases := []struct {
		name           string
		input          []byte
		expectError    bool
		expectedExists bool
	}{
		{"Valid JSON", []byte(`{"key":"value"}`), false, true},
		{"Empty JSON object", []byte(`{}`), false, true},
		{"JSON array", []byte(`[1, 2]`), false, true},
		{"JSON null", []byte(`null`), false, true},
		{"Empty byte slice", []byte{}, true, false},
		{"Nil byte slice", nil, true, false},
		{"Invalid JSON", []byte(`{"key": "value"`), true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := extractor.SetData(tc.input)

			if tc.expectError {
				assert.Error(t, err, "Expected an error for invalid input")
				assert.False(t, extractor.parsedData.Exists(), "Parsed data should not exist on error")
			} else {
				assert.NoError(t, err, "Did not expect an error for valid input")
				assert.Equal(t, tc.expectedExists, extractor.parsedData.Exists(), "Parsed data existence does not match expectation")
			}
		})
	}
}

func TestExtractContent(t *testing.T) {
	jsonData := []byte(`{
        "request": {
            "prompt": "Hello, world!",
            "id": 123
        },
        "response": {
            "answer": "Hi there!",
            "details": null
        },
        "stream_chunk": {
            "text": "This is a stream part."
        },
        "not_a_string": 42,
        "is_bool": true,
        "an_object": {"a": 1}
    }`)

	testCases := []struct {
		name            string
		config          *aicontentsecurity.GjsonConfig
		data            []byte
		methodToTest    func(*GjsonContentExtractor) string
		expectedContent string
	}{
		// --- Request Content ---
		{"Req: Normal", &aicontentsecurity.GjsonConfig{RequestContentPath: "request.prompt"}, jsonData, (*GjsonContentExtractor).RequestContent, "Hello, world!"},
		{"Req: Path not found", &aicontentsecurity.GjsonConfig{RequestContentPath: "request.nonexistent"}, jsonData, (*GjsonContentExtractor).RequestContent, ""},
		{"Req: Empty path", &aicontentsecurity.GjsonConfig{RequestContentPath: ""}, jsonData, (*GjsonContentExtractor).RequestContent, ""},
		{"Req: Nil config", nil, jsonData, (*GjsonContentExtractor).RequestContent, ""},
		// --- Response Content ---
		{"Resp: Normal", &aicontentsecurity.GjsonConfig{ResponseContentPath: "response.answer"}, jsonData, (*GjsonContentExtractor).ResponseContent, "Hi there!"},
		{"Resp: Value is null", &aicontentsecurity.GjsonConfig{ResponseContentPath: "response.details"}, jsonData, (*GjsonContentExtractor).ResponseContent, ""},
		// --- Stream Response Content ---
		{"Stream: Normal", &aicontentsecurity.GjsonConfig{StreamResponseContentPath: "stream_chunk.text"}, jsonData, (*GjsonContentExtractor).StreamResponseContent, "This is a stream part."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractor := NewGjsonContentExtractor(tc.config)
			if tc.data != nil {
				err := extractor.SetData(tc.data)
				assert.NoError(t, err)
			}

			content := tc.methodToTest(extractor)
			assert.Equal(t, tc.expectedContent, content)
		})
	}
}

func TestExtractIDFromHeaders(t *testing.T) {
	testCases := []struct {
		name          string
		config        *aicontentsecurity.GjsonConfig
		headersToAdd  map[string]string
		initialIDMap  map[string]string
		expectedIDMap map[string]string
		shouldPanic   bool
	}{
		{
			name: "Normal extraction",
			config: &aicontentsecurity.GjsonConfig{HeaderFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "X-User-ID", TargetField: "user_id"},
				{SourceField: "X-Request-ID", TargetField: "req_id"},
			}},
			headersToAdd:  map[string]string{"X-User-ID": "u1", "X-Request-ID": "r1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{"user_id": "u1", "req_id": "r1"},
		},
		{
			name:          "Nil config",
			config:        nil,
			headersToAdd:  map[string]string{"X-User-ID": "u1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{},
		},
		{
			name:          "Nil HeaderFields in config",
			config:        &aicontentsecurity.GjsonConfig{HeaderFields: nil},
			headersToAdd:  map[string]string{"X-User-ID": "u1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{},
		},
		{
			name: "Partially missing headers",
			config: &aicontentsecurity.GjsonConfig{HeaderFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "X-User-ID", TargetField: "user_id"},
				{SourceField: "X-Missing", TargetField: "missing"},
			}},
			headersToAdd:  map[string]string{"X-User-ID": "u1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{"user_id": "u1"},
		},
		{
			name: "Empty source or target fields",
			config: &aicontentsecurity.GjsonConfig{HeaderFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "", TargetField: "id1"},
				{SourceField: "X-User-ID", TargetField: ""},
			}},
			headersToAdd:  map[string]string{"X-User-ID": "u1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{},
		},
		{
			name: "Case-insensitive header names (SourceField is lowercase)",
			config: &aicontentsecurity.GjsonConfig{HeaderFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "x-user-id", TargetField: "user_id"},
			}},
			headersToAdd:  map[string]string{"X-User-ID": "u1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{"user_id": "u1"},
		},
		{
			name: "Existing idMap gets appended to",
			config: &aicontentsecurity.GjsonConfig{HeaderFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "X-Request-ID", TargetField: "req_id"},
			}},
			headersToAdd:  map[string]string{"X-Request-ID": "r1"},
			initialIDMap:  map[string]string{"existing": "val"},
			expectedIDMap: map[string]string{"existing": "val", "req_id": "r1"},
		},
		{
			name:          "Panic on nil element in slice",
			config:        &aicontentsecurity.GjsonConfig{HeaderFields: []*aicontentsecurity.FieldMapping{nil}},
			headersToAdd:  map[string]string{"X-User-ID": "u1"},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{},
			shouldPanic:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractor := NewGjsonContentExtractor(tc.config)
			idMap := tc.initialIDMap

			headers := envoy.NewRequestHeaderMap(http.Header{})
			for k, v := range tc.headersToAdd {
				headers.Set(k, v)
			}

			if tc.shouldPanic {
				assert.Panics(t, func() {
					extractor.IDsFromRequestHeaders(headers, idMap)
				}, "Should panic on nil FieldMapping due to direct field access")
				return
			}

			extractor.IDsFromRequestHeaders(headers, idMap)
			assert.Equal(t, tc.expectedIDMap, idMap)
		})
	}

}

func TestExtractIDFromData(t *testing.T) {
	jsonData := []byte(`{
        "user": {"id": "user-body", "name": "John"},
        "request_id": "req-body",
        "nested": {"deep": {"session_id": 12345}},
        "complex": { "a": 1, "b": 2 }
    }`)

	testCases := []struct {
		name          string
		config        *aicontentsecurity.GjsonConfig
		initialIDMap  map[string]string
		expectedIDMap map[string]string
	}{
		// MODIFIED: Merged multiple extraction tests into one comprehensive case.
		{
			name: "Comprehensive extraction of various types",
			config: &aicontentsecurity.GjsonConfig{BodyFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "user.id", TargetField: "user_id"},                   // Normal
				{SourceField: "request_id", TargetField: "req_id"},                 // Normal
				{SourceField: "nested.deep.session_id", TargetField: "session_id"}, // Nested + Number
				{SourceField: "complex", TargetField: "complex_field"},             // Complex Object
			}},
			initialIDMap: make(map[string]string),
			expectedIDMap: map[string]string{
				"user_id":       "user-body",
				"req_id":        "req-body",
				"session_id":    "12345",
				"complex_field": `{ "a": 1, "b": 2 }`,
			},
		},
		{
			name:          "Nil config",
			config:        nil,
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{},
		},
		{
			name: "Nil element in slice is handled gracefully",
			config: &aicontentsecurity.GjsonConfig{BodyFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "user.id", TargetField: "user_id"},
				nil,
				{SourceField: "request_id", TargetField: "req_id"},
			}},
			initialIDMap:  make(map[string]string),
			expectedIDMap: map[string]string{"user_id": "user-body", "req_id": "req-body"},
		},
		{
			name: "Existing idMap gets appended to",
			config: &aicontentsecurity.GjsonConfig{BodyFields: []*aicontentsecurity.FieldMapping{
				{SourceField: "request_id", TargetField: "req_id"},
			}},
			initialIDMap:  map[string]string{"existing": "val"},
			expectedIDMap: map[string]string{"existing": "val", "req_id": "req-body"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractor := NewGjsonContentExtractor(tc.config)
			err := extractor.SetData(jsonData)
			assert.NoError(t, err)

			idMap := tc.initialIDMap
			extractor.IDsFromRequestData(idMap)
			assert.Equal(t, tc.expectedIDMap, idMap)
		})
	}

	t.Run("No data set", func(t *testing.T) {
		config := &aicontentsecurity.GjsonConfig{BodyFields: []*aicontentsecurity.FieldMapping{{SourceField: "user.id", TargetField: "user_id"}}}
		extractor := NewGjsonContentExtractor(config)
		idMap := make(map[string]string)
		extractor.IDsFromRequestData(idMap)
		assert.Empty(t, idMap, "idMap should be empty if no data is set")
	})

	t.Run("Panic on nil idMap", func(t *testing.T) {
		config := &aicontentsecurity.GjsonConfig{BodyFields: []*aicontentsecurity.FieldMapping{{SourceField: "user.id", TargetField: "user_id"}}}
		extractor := NewGjsonContentExtractor(config)
		_ = extractor.SetData(jsonData)
		assert.Panics(t, func() {
			extractor.IDsFromRequestData(nil)
		}, "Should panic if idMap is nil")
	})
}
