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

package tokenizer

import (
	"encoding/json"
	"testing"
)

func TestOpenaiTokenizer_GetToken(t *testing.T) {
	tokenizer := &OpenaiTokenizer{}

	// Construct a simulated message
	messages := []OpenaiPromptMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello! How are you?"},
	}

	// Convert messages to JSON string
	data, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("failed to marshal messages: %v", err)
	}

	// Test with gpt-3.5-turbo model
	model := "gpt-3.5-turbo-0613"
	tokens, err := tokenizer.GetToken(string(data), model)
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}

	t.Logf("model=%s, tokens=%d", model, tokens)

	// Simple assertion: the token count should be greater than 0
	if tokens <= 0 {
		t.Errorf("expected tokens > 0, got %d", tokens)
	}
}
