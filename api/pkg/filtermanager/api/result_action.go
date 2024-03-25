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

package api

import "net/http"

// ResultAction is the result returned by each Filter method
type ResultAction interface {
	OK()
}

type isResultAction struct {
	typeid int // we need to add a field, otherwises Go will optimize all `&isResultAction{}` to same address.
}

func (i *isResultAction) OK() {}

var (
	// Continue indicates the process can continue without steering
	Continue ResultAction = &isResultAction{typeid: 0}
	// WaitAllData controls if the request/response body needs to be fully buffered during processing by Go plugin.
	// If this action is returned, DecodeData/EncodeData will be called by DecodeRequest/EncodeResponse.
	WaitAllData ResultAction = &isResultAction{typeid: 1}
	// LocalResponse controls if a local reply should be returned from Envoy instead of using the
	// upstream response. See comments below for how to use it.
)

// LocalResponse represents the reply sent directly to the client instead of using the
// upstream response. Return `&LocalResponse{Code: 4xx, ...}` in the method if you want
// to send such a reply.
type LocalResponse struct {
	isResultAction

	Code int
	// If the Msg is not empty, we will set the reply's body according to the Msg.
	// The rule to generate body is:
	// 1. If Content-Type is specified in the Header, the Msg will be sent directly.
	// 2. If the response header is received, and the Content-Type is "application/json", the Msg is wrapped into a JSON like `{"msg": $MSG}`.
	// 3. If the request doesn't have Content-Type or the Content-Type is "application/json", the Msg is wrapped into a JSON.
	// 4. Otherwise, the Msg will be sent directly.
	Msg    string
	Header http.Header
}
