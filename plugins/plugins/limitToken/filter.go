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

package limitToken

import (
	"mime"
	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/plugins/limitToken/limiter"
	"mosn.io/htnn/plugins/plugins/limitToken/sseparser"
	"net/http"
)

// factory creates a filter instance by binding the configuration and callback.
// During initialization, it also creates a buffer to store content chunks for moderation.
func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	config := c.(*config)
	return &filter{
		callbacks: callbacks,
		config:    config,
		sseParser: sseparser.NewStreamEventParser(),
		limiter: limiter.NewLimiter(
			limiter.WithRedisLimiter(config.rdb),
			limiter.WithRegexps(config.regexps),
			limiter.WithRejectedMsg(config.RejectedMsg),
			limiter.WithRejectedCode(int(config.RejectedCode)),
			limiter.WithTokenizer(config.Tokenizer),
			limiter.WithTokenStats(config.tokenStats),
		),
		bodyBuffer: []byte{},
	}
}

// filter implements request and response interception and content moderation.
// Includes:
//   - Extract content from requests/responses
//   - Buffer content chunks
//   - Call AI moderation service
//   - Intercept content that violates rules
type filter struct {
	api.PassThroughFilter

	callbacks      api.FilterCallbackHandler
	config         *config
	streamResponse bool // Whether response is streaming

	sseParser  *sseparser.StreamEventParser // SSE event parser
	limiter    *limiter.Limiter
	bodyBuffer []byte // Buffer for non-streaming response data

	streamCloseFlag bool // Stream close flag, set to true when violation detected
}

// isStream checks whether the response is of SSE (Server-Sent Events) type.
func isStream(headers api.HeaderMap) bool {
	contentType, ok := headers.Get("Content-Type")
	if !ok {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "text/event-stream"
}

// DecodeRequest intercepts request data for moderation
func (f *filter) DecodeRequest(headers api.RequestHeaderMap, data api.BufferInstance, trailers api.RequestTrailerMap) api.ResultAction {
	return f.decodeDataHandler(headers, data, true)
}

// EncodeHeaders checks if response headers indicate streaming data.
// If streaming, initialize SSE parser.
func (f *filter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	if isStream(headers) && !endStream {
		f.sseParser = sseparser.NewStreamEventParser()
		if !f.config.StreamingEnabled {
			// If streaming moderation is disabled, wait for all data before processing
			return api.WaitAllData
		}
		f.streamResponse = true
	}
	return api.Continue
}

// EncodeData handles response body data, supporting both streaming and non-streaming.
func (f *filter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	if f.streamResponse {
		return f.streamDataHandler(data, endStream)
	} else {
		return f.encodeDataHandler(data, endStream)
	}
}

// EncodeResponse handles final processing for streaming responses.
func (f *filter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	return f.streamDataHandler(data, true)
}

// decodeDataHandler handles non-streaming request/response body.
// Core logic:
//   - Handle chunked or full body
//   - Extract ID and content
//   - Write to buffer and call moderation service
//   - Block if violation, else pass original data
func (f *filter) decodeDataHandler(headers api.RequestHeaderMap, data api.BufferInstance, endStream bool) api.ResultAction {
	extractor := f.config.extractor

	if len(f.bodyBuffer) == 0 && endStream {
		// Single full body
		err := extractor.SetData(data.Bytes())
		if err != nil {
			api.LogErrorf("failed to set data to extractor with original data:%s err: %v", data.String(), err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
	} else {
		// Chunked data
		if f.bodyBuffer == nil {
			f.bodyBuffer = make([]byte, 0, 2048)
		}
		f.bodyBuffer = append(f.bodyBuffer, data.Bytes()...)
		data.Reset()

		if !endStream {
			return api.Continue
		}

		// Full data collected, set to extractor
		err := extractor.SetData(f.bodyBuffer)
		if err != nil {
			api.LogErrorf("failed to set bodyBuffer to extractor with original data:%s err: %v", f.bodyBuffer, err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
	}

	content, model := extractor.RequestContentAndModel()
	return f.limiter.DecodeData(headers, f.config.Rule, content, model)
}

// encodeDataHandler processes non-streaming response data
func (f *filter) encodeDataHandler(data api.BufferInstance, endStream bool) api.ResultAction {
	extractor := f.config.extractor

	if len(f.bodyBuffer) == 0 && endStream {
		// Single full body
		err := extractor.SetData(data.Bytes())
		if err != nil {
			api.LogErrorf("failed to set data to extractor with original data:%s err: %v", data.String(), err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
	} else {
		// Chunked data
		if f.bodyBuffer == nil {
			f.bodyBuffer = make([]byte, 0, 2048)
		}
		f.bodyBuffer = append(f.bodyBuffer, data.Bytes()...)
		data.Reset()

		if !endStream {
			return api.Continue
		}

		// Full data collected, set to extractor
		err := extractor.SetData(f.bodyBuffer)
		if err != nil {
			api.LogErrorf("failed to set bodyBuffer to extractor with original data:%s err: %v", f.bodyBuffer, err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
	}

	content, model, completeToken, promptToken := extractor.ResponseContentAndModel()
	return f.limiter.EncodeData(content, model, int(completeToken), int(promptToken))
}

// streamDataHandler processes streaming response data (SSE)
func (f *filter) streamDataHandler(data api.BufferInstance, endStream bool) api.ResultAction {
	extractor := f.config.extractor

	if f.streamCloseFlag {
		return &api.LocalResponse{Code: http.StatusBadGateway}
	}

	// Feed new data to SSE parser
	f.sseParser.Append(data.Bytes())
	data.Reset()

	// Parse events one by one
	newAddedEventFlag := true
	for {
		event, err := f.sseParser.TryParse()
		if err != nil {
			api.LogErrorf("SSE parsing error: %v", err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
		if event == nil {
			break
		}
		content, model := extractor.StreamResponseContentAndModel()
		f.limiter.EncodeStreamData(content, model, false)
		newAddedEventFlag = false
	}

	// If no new event and stream not finished, continue waiting
	if !endStream && newAddedEventFlag {
		return api.Continue
	}

	// Flush buffer when stream ends
	if endStream {
		f.limiter.EncodeStreamData("", "", true)
	}

	return api.Continue
}
