package plugins

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"mosn.io/moe/pkg/filtermanager"
	"mosn.io/moe/tests/integration/plugins/data_plane"
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
			controlPlane.UseGoPluginConfig(tt.config)
			resp, err := dp.Get("/echo", nil)
			assert.Nil(t, err)
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
			assert.Nil(t, err)
			defer resp.Body.Close()
			tt.expectWithBody(t, resp)
		})
	}
}

func assertBodyHas(t *testing.T, exp string, resp *http.Response) {
	d, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(d), exp)
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
			controlPlane.UseGoPluginConfig(tt.config)
			resp, err := dp.Get("/echo", nil)
			assert.Nil(t, err)
			tt.expectWithoutBody(t, resp)

			hdr := http.Header{}
			resp, err = dp.Post("/slow_resp", hdr, strings.NewReader(strings.Repeat("01", 1024)))
			assert.Nil(t, err)
			defer resp.Body.Close()
			tt.expectWithBody(t, resp)
		})
	}
}
