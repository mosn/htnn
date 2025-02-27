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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
)

func TestOIDC(t *testing.T) {
	dp, err := dataplane.StartDataPlane(t, &dataplane.Option{})
	if err != nil {
		t.Fatalf("failed to start data plane: %v", err)
		return
	}
	defer dp.Stop()

	helper.WaitServiceUp(t, ":4444", "hydra")

	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/echo", dp.Port())
	hydraCmd := "hydra create client --response-type code,id_token " +
		"--grant-type authorization_code,refresh_token -e http://127.0.0.1:4445 " +
		"--redirect-uri " + redirectURL + " --format json"
	cmdline := "docker compose -f ./testdata/services/docker-compose.yml " +
		"exec --no-TTY hydra " + hydraCmd
	cmds := strings.Fields(cmdline)
	cmd := exec.Command(cmds[0], cmds[1:]...)
	stdout, err := cmd.Output()
	if err != nil {
		reason := string(err.(*exec.ExitError).Stderr)
		require.NoError(t, err, reason)
	}
	t.Logf("hydra output: %s", stdout)

	type hydraOutput struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}

	var hydra hydraOutput
	json.Unmarshal(stdout, &hydra)

	config := controlplane.NewSinglePluginConfig("oidc", map[string]interface{}{
		"clientId":     hydra.ClientID,
		"clientSecret": hydra.ClientSecret,
		"redirectUrl":  redirectURL,
		"issuer":       "http://hydra:4444",
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	uri := ""
	var resp *http.Response
	require.Eventually(t, func() bool {
		resp, err = dp.Get("/echo?a=1", nil)
		require.Nil(t, err)
		uri = resp.Header.Get("Location")
		return uri != ""
	}, 15*time.Second, 1*time.Second)

	u, err := url.ParseRequestURI(uri)
	require.NoError(t, err)
	require.Equal(t, "hydra:4444", u.Host)
	require.Equal(t, hydra.ClientID, u.Query().Get("client_id"))
	require.Equal(t, redirectURL, u.Query().Get("redirect_uri"))
	encodedURL := strings.Split(u.Query().Get("state"), ".")[1]
	b, _ := base64.URLEncoding.DecodeString(encodedURL)
	originURL := string(b)
	require.Equal(t, fmt.Sprintf("http://localhost:%d/echo?a=1", dp.Port()), originURL)
	require.NotEmpty(t, u.Query().Get("nonce"))
	require.NotEmpty(t, u.Query().Get("code_challenge"))
	cookie := resp.Header.Get("Set-Cookie")
	require.Regexp(t, `^htnn_oidc_nonce_[^=]+=[^;]+; Max-Age=3600; HttpOnly$`, cookie)

	// the request is sent from the host
	uri = strings.Replace(uri, "http://hydra:4444", "http://127.0.0.1:4444", 1)
	req, err := http.NewRequest("GET", uri, nil)
	require.NoError(t, err)

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 302, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Location"), "http://127.0.0.1:3000/login")
}
