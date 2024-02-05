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

package ext_auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConfTimeout(t *testing.T) {
	s := `{"httpService":{
		"timeout": "10s"
	}}`
	conf := &config{}
	protojson.Unmarshal([]byte(s), conf)
	conf.Init(nil)
	assert.Equal(t, 10*time.Second, conf.client.Timeout)
}

func TestBadConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "at least one config type is required",
			input: `{}`,
			err:   "value is required",
		},
		{
			name:  "invalid HttpService.Url",
			input: `{"httpService":{"url":"127.0.0.1"}}`,
			err:   "invalid HttpService.Url: value must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config{}
			err := protojson.Unmarshal([]byte(tt.input), conf)
			if err == nil {
				err = conf.Validate()
			}
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}
