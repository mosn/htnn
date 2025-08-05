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

package localservice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func TestRequestModeration_live(t *testing.T) {
	t.Skip("Skipping live test to avoid dependency on external services")
	service, err := New(&aicontentsecurity.LocalModerationServiceConfig{
		BaseUrl:            "http://127.0.0.1:10902",
		UnhealthyWords:     []string{"hate", "violence"},
		CustomErrorMessage: "Content blocked due to policy violation.",
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantAllow bool
	}{
		{
			name:      "safe content",
			content:   "This is a normal message",
			wantErr:   false,
			wantAllow: true,
		},
		{
			name:      "unsafe content with 'hate'",
			content:   "This message contains hate speech.",
			wantErr:   false,
			wantAllow: false,
		},
		{
			name:      "unsafe content with 'violence'",
			content:   "This is a story about violence.",
			wantErr:   false,
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.Request(ctx, tt.content, make(map[string]string))
			if (err != nil) != tt.wantErr {
				t.Errorf("Request() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && result.Allow != tt.wantAllow {
				t.Errorf("Request() result.Allow = %v, want %v. Reason: %s", result.Allow, tt.wantAllow, result.Reason)
			}
		})
	}
}
