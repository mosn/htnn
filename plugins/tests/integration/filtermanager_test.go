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

package integration

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/plugins/tests/integration/data_plane"
)

func assertBody(t *testing.T, exp string, resp *http.Response) {
	d, _ := io.ReadAll(resp.Body)
	assert.Equal(t, exp, string(d))
}

func TestFilterManagerDecode(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	b := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
		},
	}
	sThenB := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
		},
	}
	sThenBThenS := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
		},
	}
	sThenBThenSThenB := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
		},
	}
	nbThenS := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
		},
	}
	bThenNb := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
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
			name:   "buffer",
			config: b,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Echo-Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "01buffer\n", resp)
			},
		},
		{
			name:   "stream then buffer",
			config: sThenB,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer"}, resp.Header.Values("Echo-Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "01stream\nbuffer\n", resp)
			},
		},
		{
			name:   "stream then buffer then stream",
			config: sThenBThenS,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream"}, resp.Header.Values("Echo-Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "01stream\nbuffer\nstream\n", resp)
			},
		},
		{
			name:   "stream then buffer then stream then buffer",
			config: sThenBThenSThenB,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream", "buffer"}, resp.Header.Values("Echo-Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream", "buffer"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "01stream\nbuffer\nstream\nbuffer\n", resp)
			},
		},
		{
			name:   "no buffer then stream",
			config: nbThenS,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"no buffer", "stream"}, resp.Header.Values("Echo-Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"no buffer", "stream"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "0no buffer\nstream\n1no buffer\nstream\nno buffer\nstream\n", resp)
			},
		},
		{
			name:   "buffer then no buffer",
			config: bThenNb,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer", "no buffer"}, resp.Header.Values("Echo-Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer", "no buffer"}, resp.Header.Values("Echo-Run"))
				assertBody(t, "01buffer\nno buffer\n", resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config, dp)
			resp, err := dp.Get("/echo", nil)
			require.Nil(t, err)
			tt.expectWithoutBody(t, resp)

			rd, wt := io.Pipe()
			go func() {
				for i := 0; i < 2; i++ {
					time.Sleep(20 * time.Millisecond)
					_, err := wt.Write([]byte(strconv.Itoa(i)))
					assert.Nil(t, err)
				}
				wt.Close()
			}()
			resp, err = dp.Post("/echo", nil, rd)
			require.Nil(t, err)
			defer resp.Body.Close()
			tt.expectWithBody(t, resp)
		})
	}
}

func assertBodyHas(t *testing.T, exp string, resp *http.Response) {
	d, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(d), exp)
	// set the body back so the next assertion can read the body
	resp.Body = io.NopCloser(bytes.NewBuffer(d))
}

func TestFilterManagerEncode(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	b := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
		},
	}
	sThenB := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
		},
	}
	sThenBThenS := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
		},
	}
	sThenBThenSThenB := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
		},
	}
	nbThenS := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
				},
			},
		},
	}
	bThenNb := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
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
			name:   "buffer",
			config: b,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer"}, resp.Header.Values("Run"))
				assertBodyHas(t, "01buffer\n", resp)
			},
		},
		{
			name:   "stream then buffer",
			config: sThenB,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer"}, resp.Header.Values("Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer"}, resp.Header.Values("Run"))
				assertBodyHas(t, "01stream\nbuffer\n", resp)
			},
		},
		{
			name:   "stream then buffer then stream",
			config: sThenBThenS,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream"}, resp.Header.Values("Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream"}, resp.Header.Values("Run"))
				assertBodyHas(t, "01stream\nbuffer\nstream\n", resp)
			},
		},
		{
			name:   "stream then buffer then stream then buffer",
			config: sThenBThenSThenB,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream", "buffer"}, resp.Header.Values("Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream", "buffer", "stream", "buffer"}, resp.Header.Values("Run"))
				assertBodyHas(t, "01stream\nbuffer\nstream\nbuffer\n", resp)
			},
		},
		{
			name:   "no buffer then stream",
			config: nbThenS,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"no buffer", "stream"}, resp.Header.Values("Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"no buffer", "stream"}, resp.Header.Values("Run"))
				assertBodyHas(t, "01no buffer\nstream\n01", resp)
			},
		},
		{
			name:   "buffer then no buffer",
			config: bThenNb,
			expectWithoutBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer", "no buffer"}, resp.Header.Values("Run"))
			},
			expectWithBody: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"buffer", "no buffer"}, resp.Header.Values("Run"))
				assertBodyHas(t, "01buffer\nno buffer\n", resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config, dp)
			resp, err := dp.Get("/echo", nil)
			require.Nil(t, err)
			tt.expectWithoutBody(t, resp)

			hdr := http.Header{}
			resp, err = dp.Post("/slow_resp", hdr, strings.NewReader(strings.Repeat("01", 1024)))
			require.Nil(t, err)
			defer resp.Body.Close()
			tt.expectWithBody(t, resp)
		})
	}
}

func TestFilterManagerDecodeLocalReply(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	dh := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Decode:  true,
					Headers: true,
				},
			},
		},
	}
	dd := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Data:   true,
				},
			},
		},
	}
	ddThenB := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Data:   true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
		},
	}
	dr := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
		},
	}
	bThenDh := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "localReply",
				Config: &Config{
					Decode:  true,
					Headers: true,
				},
			},
		},
	}
	bThenDd := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Data:   true,
				},
			},
		},
	}

	lrThenE := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Data:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
		},
	}
	fOrder := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Data:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
			// should local reply in DecodeData after running all DecodeHeaders
		},
	}
	fOrderM := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "localReply",
				Config: &Config{
					Decode: true,
					Data:   true,
				},
			},
			// should local reply in DecodeData before DecodeRequest
			{
				Name: "buffer",
				Config: &Config{
					Decode: true,
					Need:   true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Decode: true,
				},
			},
		},
	}

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		expect func(t *testing.T, resp *http.Response)
	}{
		{
			name:   "DecodeHeaders",
			config: dh,
		},
		{
			name:   "DecodeData",
			config: dd,
		},
		{
			name:   "DecodeData before DecodeRequest",
			config: ddThenB,
		},
		{
			name:   "DecodeRequest",
			config: dr,
		},
		{
			name:   "DecodeHeaders after DecodeRequest",
			config: bThenDh,
		},
		{
			name:   "DecodeData after DecodeRequest",
			config: bThenDd,
		},
		{
			name:   "LocalReply rewritten by Encode",
			config: lrThenE,
			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, []string{"stream"}, resp.Header.Values("Run"))
				assertBodyHas(t, "stream\n", resp)
			},
		},
		{
			name:   "Ensure the header filters' order after DecodeRequest",
			config: fOrder,
			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, "buffer|stream", resp.Header.Get("Order"))
			},
		},
		{
			name:   "Ensure the header filters' order between multiple DecodeRequest",
			config: fOrderM,
			expect: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, "buffer", resp.Header.Get("Order"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config, dp)
			resp, err := dp.Post("/echo", nil, strings.NewReader("any"))
			require.Nil(t, err)
			assert.Equal(t, 206, resp.StatusCode)
			assert.Equal(t, []string{"reply"}, resp.Header.Values("local"))
			assertBodyHas(t, "ok", resp)

			if tt.expect != nil {
				tt.expect(t, resp)
			}
		})
	}
}

func TestFilterManagerEncodeLocalReply(t *testing.T) {
	dp, err := data_plane.StartDataPlane(t, nil)
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	eh := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Encode:  true,
					Headers: true,
				},
			},
		},
	}
	ed := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Encode: true,
					Data:   true,
				},
			},
		},
	}
	er := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
		},
	}
	edThenB := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
			{
				Name: "localReply",
				Config: &Config{
					Encode: true,
					Data:   true,
				},
			},
		},
	}
	bThenEh := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Encode:  true,
					Headers: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
		},
	}
	bThenEd := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Encode: true,
					Data:   true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
		},
	}
	bThenSThenEh := &filtermanager.FilterManagerConfig{
		Plugins: []*filtermanager.FilterConfig{
			{
				Name: "localReply",
				Config: &Config{
					Encode:  true,
					Headers: true,
				},
			},
			{
				Name: "stream",
				Config: &Config{
					Encode: true,
				},
			},
			{
				Name: "buffer",
				Config: &Config{
					Encode: true,
					Need:   true,
				},
			},
		},
	}

	tests := []struct {
		name   string
		config *filtermanager.FilterManagerConfig
		expect func(t *testing.T, resp *http.Response)
	}{
		{
			name:   "EncodeHeaders",
			config: eh,
		},
		{
			name:   "EncodeData",
			config: ed,
		},
		{
			name:   "EncodeResponse",
			config: er,
		},
		{
			name:   "EncodeData before EncodeResponse",
			config: edThenB,
		},
		{
			name:   "EncodeHeaders after EncodeResponse",
			config: bThenEh,
		},
		{
			name:   "EncodeData after EncodeResponse",
			config: bThenEd,
		},
		{
			name:   "Buffer all, then run header filters from stream and local reply",
			config: bThenSThenEh,
			expect: func(t *testing.T, resp *http.Response) {
				// only EncodeData in localReply is run
				assertBody(t, "ok", resp)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPlane.UseGoPluginConfig(tt.config, dp)
			hdr := http.Header{}
			hdr.Add("from", "reply")
			resp, err := dp.Post("/echo", hdr, strings.NewReader("any"))
			require.Nil(t, err)
			assert.Equal(t, 206, resp.StatusCode)
			assert.Equal(t, "reply", resp.Header.Get("local"))
			assertBodyHas(t, "ok", resp)

			if tt.expect != nil {
				tt.expect(t, resp)
			}
		})
	}
}
