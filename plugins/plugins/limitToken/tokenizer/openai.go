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
	"github.com/pkoukk/tiktoken-go"
	"log"
	"strings"
)

type OpenaiTokenizer struct{}

type OpenaiPromptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (t *OpenaiTokenizer) GetToken(messagesStr, model string) (int, error) {

	var messages []OpenaiPromptMessage
	err := json.Unmarshal([]byte(messagesStr), &messages)
	if err != nil {
		log.Printf("unmarshal failed: %v", err)
		return 0, err
	}

	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		log.Printf("encoding for model %s: %v", model, err)
		return 0, err
	}

	var tokensPerMessage int
	switch model {
	case "gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k-0613",
		"gpt-4-0314",
		"gpt-4-32k-0314",
		"gpt-4-0613",
		"gpt-4-32k-0613":
		tokensPerMessage = 3
	case "gpt-3.5-turbo-0301":
		tokensPerMessage = 4
	default:
		if strings.Contains(model, "gpt-3.5-turbo") {
			log.Println("warning: gpt-3.5-turbo may update over time. Returning num tokens assuming gpt-3.5-turbo-0613.")
			return t.GetToken(messagesStr, "gpt-3.5-turbo-0613")
		} else if strings.Contains(model, "gpt-4") {
			log.Println("warning: gpt-4 may update over time. Returning num tokens assuming gpt-4-0613.")
			return t.GetToken(messagesStr, "gpt-4-0613")
		} else {
			err := log.Output(2, "num_tokens_from_messages() is not implemented for model "+model+". See https://github.com/openai/openai-python/blob/main/chatml.md for information.")
			return 0, err
		}
	}

	numTokens := 0
	for _, message := range messages {
		numTokens += tokensPerMessage
		numTokens += len(tkm.Encode(message.Content, nil, nil))
		numTokens += len(tkm.Encode(message.Role, nil, nil))
	}

	numTokens += 3
	return numTokens, nil
}
