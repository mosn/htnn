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

//go:build envoydev

package integration

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
)

func TestFilterManagerLogWithTrailers(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		ExpectLogPattern: []string{
			`receive request trailers: .*expires:Wed, 21 Oct 2015 07:28:00 GMT.*`,
		},
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	lp := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name:   "onLog",
				Config: &Config{},
			},
		},
	}

	controlPlane.UseGoPluginConfig(t, lp, dp)
	hdr := http.Header{}
	trailer := http.Header{}
	trailer.Add("Expires", "Wed, 21 Oct 2015 07:28:00 GMT")
	resp, err := dp.PostWithTrailer("/echo", hdr, bytes.NewReader([]byte("test")), trailer)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
