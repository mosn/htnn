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

package listenerpatch

import (
	"testing"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/matching/common_inputs/network/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name: "bad access log",
			input: `
{
    "accessLog": [{
    }]
}`,
			err: `access log config is nil`,
		},
		{
			name: "validate log format",
			input: `
{
    "accessLog": [{
        "name": "envoy.access_loggers.file",
        "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
            "logFormat": "%START_TIME%,%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%"
        }
    }]
}`,
			err: `unexpected token "%START_TIME%`,
		},
		{
			name: "validate file logger",
			input: `
{
    "accessLog": [{
        "name": "envoy.access_loggers.file",
        "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
            "logFormat": {
				"textFormat": "%START_TIME%,%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%"
			},
			"path": ""
        }
    }]
}`,
			err: `invalid FileAccessLog.Path: value length must be at least 1 runes`,
		},
		{
			name: "unknown logger",
			input: `
{
    "accessLog": [{
        "name": "envoy.access_loggers.http_file",
        "typedConfig": {
            "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.HTTPFileLogger",
            "logFormat": {
				"textFormat": "%START_TIME%,%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%"
			},
			"path": "/xx"
        }
    }]
}`,
			err: `unable to resolve`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &CustomConfig{}
			err := protojson.Unmarshal([]byte(tt.input), conf)
			if err == nil {
				err = conf.Validate()
			}
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
