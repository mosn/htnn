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

package moderation

import (
	"context"
)

type Moderator interface {

	// Request moderates the content of a request.
	Request(ctx context.Context, content string, idMap map[string]string) (*Result, error)

	// Response moderates the content of a response.
	Response(ctx context.Context, content string, idMap map[string]string) (*Result, error)
}

type Result struct {

	// Allow indicates whether the content is permitted.
	Allow bool

	// Reason provides an explanation for the moderation fail result.
	Reason string
}
