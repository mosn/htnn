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

package aicontentsecurity

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"

	"golang.org/x/sync/errgroup"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/contentbuffer"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/moderation"
	"mosn.io/htnn/plugins/plugins/aicontentsecurity/sseparser"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	config := c.(*config)
	return &filter{
		callbacks: callbacks,
		config:    config,
		idMap:     make(map[string]string),
		contentBuf: contentbuffer.NewContentBuffer(contentbuffer.WithMaxChars(int(config.ModerationCharLimit)),
			contentbuffer.WithOverlapCharNum(int(config.ModerationChunkOverlapLength))),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks      api.FilterCallbackHandler
	config         *config
	idMap          map[string]string
	streamResponse bool

	sseParser  *sseparser.StreamEventParser
	contentBuf *contentbuffer.ContentBuffer
	bodyBuffer []byte

	streamCloseFlag bool
}

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

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	f.config.extractor.IDsFromRequestHeaders(headers, f.idMap)
	return api.Continue
}

func (f *filter) DecodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	return f.dataHandler(data, endStream, false)
}

func (f *filter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	if isStream(headers) && !endStream {
		f.sseParser = sseparser.NewStreamEventParser()
		if !f.config.StreamingEnabled {
			return api.WaitAllData
		}
		f.streamResponse = true
	}
	return api.Continue
}

func (f *filter) EncodeData(data api.BufferInstance, endStream bool) api.ResultAction {
	if f.streamResponse {
		return f.streamDataHandler(data, endStream)
	} else {
		return f.dataHandler(data, endStream, true)
	}
}

func (f *filter) EncodeResponse(headers api.ResponseHeaderMap, data api.BufferInstance, trailers api.ResponseTrailerMap) api.ResultAction {
	return f.streamDataHandler(data, true)
}

func (f *filter) streamDataHandler(data api.BufferInstance, endStream bool) api.ResultAction {
	if f.streamCloseFlag {
		data.Reset()
		return &api.LocalResponse{Code: http.StatusBadGateway}
	}

	f.sseParser.Append(data.Bytes())
	data.Reset()

	// event parser
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

		newAddedEventFlag = false
		_ = f.config.extractor.SetData([]byte(event.Data))
		eventContent := f.config.extractor.StreamResponseContent()
		// Always write to ensure the counter is correct.
		f.contentBuf.Write([]byte(eventContent))
	}

	// No new complete event
	if !endStream && newAddedEventFlag {
		return api.Continue
	}

	// If the stream ends, flush the content buffer and send everything for moderation.
	if endStream {
		f.contentBuf.Flush()
	}

	// Get the chunks that have accumulated up to MaxChars, ready to be sent for moderation.
	completedResult := f.contentBuf.GetCompletedResult()
	// Consume the events accumulated in the above chunks, making them available for writing downstream.
	f.sseParser.Consume(completedResult.CompletedEvents)

	if completedResult.Chunks != nil {
		ctx, cancel := context.WithTimeout(context.Background(), f.config.moderationTimeout)
		defer cancel()

		res, err := f.performModeration(ctx, completedResult.Chunks, true)
		if err != nil {
			api.LogErrorf("Failed to perform moderation: %v", err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}

		if res != nil && !res.Allow {
			// Graceful shutdown.
			api.LogInfof("Content rejected by moderation service, reason: %s", res.Reason)
			errorPayload := fmt.Sprintf(
				"event: error\ndata: %s\n\n",
				res.Reason)
			err = data.Set([]byte(errorPayload))
			if err != nil {
				api.LogErrorf("Failed to set error payload: %v", err)
				return &api.LocalResponse{Code: http.StatusBadGateway}
			}
			f.streamCloseFlag = true
			return api.Continue
		}
	}

	// Write the events that have passed moderation and remove them from the buffer.
	err := data.Append(f.sseParser.ParsedBytes())
	if err != nil {
		api.LogErrorf("Failed to append parsed bytes: %v", err)
		return &api.LocalResponse{Code: http.StatusBadGateway}
	}
	f.sseParser.PruneParsedData()

	return api.Continue
}

type moderationBlockedError struct {
	Result *moderation.Result
}

func (e *moderationBlockedError) Error() string {
	return fmt.Sprintf("content blocked: %s", e.Result.Reason)
}

func (f *filter) performModeration(ctx context.Context, buffers []string, isEncode bool) (*moderation.Result, error) {
	if len(buffers) == 0 {
		return &moderation.Result{Allow: true}, nil
	}

	group, ctx := errgroup.WithContext(ctx)
	concurrencyLimit := 5
	sem := make(chan struct{}, concurrencyLimit)

	for _, buffer := range buffers {
		buf := buffer

		sem <- struct{}{}
		group.Go(func() error {
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			var res *moderation.Result
			var err error
			if isEncode {
				res, err = f.config.moderator.Response(ctx, buf, f.idMap)
			} else {
				res, err = f.config.moderator.Request(ctx, buf, f.idMap)
			}
			if err != nil {
				return err
			}
			if !res.Allow {
				return &moderationBlockedError{Result: res}
			}
			return nil
		})
	}

	err := group.Wait()
	if err == nil {
		return &moderation.Result{Allow: true}, nil
	}

	// moderation reject
	var blockedErr *moderationBlockedError
	if errors.As(err, &blockedErr) {
		return blockedErr.Result, nil
	}

	// other error
	return nil, err
}

func (f *filter) dataHandler(data api.BufferInstance, endStream bool, isEncode bool) api.ResultAction {
	extractor := f.config.extractor
	var err error
	actionType := "DecodeData"
	if isEncode {
		actionType = "EncodeData"
	}

	if len(f.bodyBuffer) == 0 && endStream {
		// The entire body is received in one go.
		err := extractor.SetData(data.Bytes())
		if err != nil {
			api.LogErrorf("%s failed to set data to extractor with original data:%s err: %v", actionType, data.String(), err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
	} else {
		// Packet Fragmentation
		if f.bodyBuffer == nil {
			f.bodyBuffer = make([]byte, 0, 2048)
		}
		f.bodyBuffer = append(f.bodyBuffer, data.Bytes()...)
		data.Reset()

		if !endStream {
			return api.Continue
		}

		err := extractor.SetData(f.bodyBuffer)
		// Errors are not allowed here.
		if err != nil {
			api.LogErrorf("%s failed to set bodyBuffer to extractor with original data:%s err: %v", actionType, f.bodyBuffer, err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
	}

	extractor.IDsFromRequestData(f.idMap)
	var content string
	if isEncode {
		content = extractor.ResponseContent()
	} else {
		content = extractor.RequestContent()
	}

	f.contentBuf.Write([]byte(content))
	f.contentBuf.Flush()
	contents := f.contentBuf.GetCompletedResult()

	ctx, cancel := context.WithTimeout(context.Background(), f.config.moderationTimeout)
	defer cancel()
	res, err := f.performModeration(ctx, contents.Chunks, isEncode)
	if err != nil {
		api.LogErrorf("%s moderation failed: %v", actionType, err)
		return &api.LocalResponse{Code: http.StatusBadGateway}
	}

	if res != nil && !res.Allow {
		return &api.LocalResponse{Code: http.StatusBadGateway, Msg: res.Reason}
	}

	if len(f.bodyBuffer) > 0 {
		err := data.Set(f.bodyBuffer)
		if err != nil {
			api.LogErrorf("%s failed to set processed buffer data back to response: %v", actionType, err)
			return &api.LocalResponse{Code: http.StatusBadGateway}
		}
		f.bodyBuffer = nil
	}
	return api.Continue
}
