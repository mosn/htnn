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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/types/plugins/aicontentsecurity"
)

func TestDefaultValue(t *testing.T) {
	cfg := &config{}
	cfg.CustomConfig.Config = aicontentsecurity.Config{
		ModerationTimeout:            int64(2 * time.Second / time.Millisecond), // 2000 ms
		StreamingEnabled:             true,
		ModerationCharLimit:          5000,
		ModerationChunkOverlapLength: 100,
		ExtractorConfig: &aicontentsecurity.Config_GjsonConfig{
			GjsonConfig: &aicontentsecurity.GjsonConfig{
				StreamResponseContentPath: "TEST",
				ResponseContentPath:       "TEST",
				RequestContentPath:        "TEST",
			},
		},
		ProviderConfig: &aicontentsecurity.Config_AliyunConfig{
			AliyunConfig: &aicontentsecurity.AliyunConfig{
				AccessKeyId:     "your_access_key_id",
				AccessKeySecret: "your_access_key_secret",
				MaxRiskLevel:    "high",
			},
		},
	}

	err := cfg.Init(nil)
	require.NoError(t, err, "cfg.Init() should not return an error")
}
