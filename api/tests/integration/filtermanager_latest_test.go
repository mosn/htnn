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

//go:build !envoy1.29

package integration

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/pkg/filtermanager"
	"mosn.io/htnn/api/pkg/filtermanager/model"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

func TestFilterManagerTrailers(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	s := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Decode:   true,
					Encode:   true,
					Trailers: true,
				},
			},
		},
	}
	lr := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Decode:   true,
					Trailers: true,
				},
			},
		},
	}

	tests := []struct {
		name              string
		config            *filtermanager.FilterManagerConfig
		expectWithoutBody func(t *testing.T, resp *http.Response)
		expectWithBody    func(t *testing.T, resp *http.Response)
	}{
		{
			name:   "DecodeTrailers",
			config: s,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream"}, resp.Header.Values("Echo-Trailer-Run"))
			},
		},
		{
			name:   "localReply",
			config: lr,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, 206, resp.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			hdr := http.Header{}
			trailer := http.Header{}
			trailer.Add("Expires", "Wed, 21 Oct 2015 07:28:00 GMT")
			resp, err := dp.PostWithTrailer("/echo", hdr, bytes.NewReader([]byte("test")), trailer)
			require.Nil(t, err)
			tt.expectWithoutBody(t, resp)
		})
	}
}

func TestFilterManagerBufferingWithTrailers(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		LogLevel: "debug",
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	b := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode:     true,
					NeedBuffer: true,
				},
			},
		},
	}
	bThenb := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode:     true,
					NeedBuffer: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode:     true,
					NeedBuffer: true,
				},
			},
		},
	}
	sThenbThennbThenb := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Decode:     true,
					NeedBuffer: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode:     true,
					NeedBuffer: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode:     true,
					NeedBuffer: true,
				},
			},
		},
	}

	tests := []struct {
		name           string
		config         *filtermanager.FilterManagerConfig
		expectWithBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:   "DecodeRequest",
			config: b,
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Echo-Trailer-Run"))
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "testbuffer\n", resp)
			},
		},
		{
			name:   "DecodeRequest, then DecodeRequest",
			config: bThenb,
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer", "buffer"}, resp.Header.Values("Echo-Trailer-Run"))
				assert.Equal(t, []string{"buffer", "buffer"}, resp.Header.Values("Echo-Run"))
			},
		},
		{
			name:   "DecodeTrailers, DecodeRequest, DecodeTrailers, then DecodeRequest",
			config: sThenbThennbThenb,
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "no buffer", "buffer"}, resp.Header.Values("Echo-Trailer-Run"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			hdr := http.Header{}
			trailer := http.Header{}
			trailer.Add("Expires", "Wed, 21 Oct 2015 07:28:00 GMT")
			resp, err := dp.PostWithTrailer("/echo", hdr, bytes.NewReader([]byte("test")), trailer)
			require.Nil(t, err)
			defer resp.Body.Close()
			tt.expectWithBody(t, resp)
		})
	}
}

func grpcurl(dp *dataplane.DataPlane, fullMethodName, req string) ([]byte, error) {
	prefix := "api.tests.integration.testdata.services.grpc."
	pwd, _ := os.Getwd()
	return dp.Grpcurl(filepath.Join(pwd, "testdata/services/grpc"), "sample.proto", prefix+fullMethodName, req)
}

func TestFilterManagerTrailersWithGrpcBackend(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{
		LogLevel: "debug",
	})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":50051", "grpc")

	s := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name:   "stream",
				Config: &Config{},
			},
		},
	}

	b := &filtermanager.FilterManagerConfig{
		Plugins: []*model.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					NeedBuffer: true,
					InGrpcMode: true,
					Encode:     true,
				},
			},
		},
	}

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		expect func(t *testing.T, resp []byte)
	}{
		{
			name:   "EncodeTrailers",
			config: s,
			expect: func(t *testing.T, resp []byte) {
				exp := `Response contents:
{
  "message": "Hello Jordan"
}

Response trailers received:
run: stream`
				assert.Contains(t, string(resp), exp, "response: %s", string(resp))
			},
		},
		{
			name:   "EncodeResponse",
			config: b,
			expect: func(t *testing.T, resp []byte) {
				exp := `Response contents:
{
  "message": "Hello Jordan"
}

Response trailers received:
(empty)`
				assert.Contains(t, string(resp), exp, "response: %s", string(resp))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(t, tt.config, dp)
			resp, _ := grpcurl(dp, "Sample.SayHello", `{"name":"Jordan"}`)
			tt.expect(t, resp)
		})
	}
}
