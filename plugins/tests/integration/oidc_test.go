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
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/stretchr/testify/require"

	"mosn.io/htnn/api/plugins/tests/integration/controlplane"
	"mosn.io/htnn/api/plugins/tests/integration/dataplane"
	"mosn.io/htnn/api/plugins/tests/integration/helper"
	"mosn.io/htnn/plugins/plugins/oidc"
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

	cookieEncryptionKey := "0123456789abcdef0123456789abcdef"
	config := controlplane.NewSinglePluginConfig("oidc", map[string]interface{}{
		"clientId":              hydra.ClientID,
		"clientSecret":          hydra.ClientSecret,
		"redirectUrl":           redirectURL,
		"issuer":                "http://hydra:4444",
		"enableUserinfoSupport": true,
		"userinfoFormat":        "RAW_JSON",
		"userinfoHeader":        "x-userinfo-data",
		"cookieEncryptionKey":   cookieEncryptionKey,
		"scopes":                []string{"offline_access"},
	})
	controlPlane.UseGoPluginConfig(t, config, dp)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}

	authRedirectURL := ""
	var initialResp *http.Response
	require.Eventually(t, func() bool {
		initialResp, err = client.Get(fmt.Sprintf("http://127.0.0.1:%d/echo?a=1", dp.Port()))
		require.Nil(t, err)
		authRedirectURL = initialResp.Header.Get("Location")
		return authRedirectURL != ""
	}, 15*time.Second, 1*time.Second)

	u, err := url.ParseRequestURI(authRedirectURL)
	require.NoError(t, err)
	require.Equal(t, "hydra:4444", u.Host)
	require.Equal(t, hydra.ClientID, u.Query().Get("client_id"))
	require.Equal(t, redirectURL, u.Query().Get("redirect_uri"))
	encodedURL := strings.Split(u.Query().Get("state"), ".")[1]
	b, _ := base64.URLEncoding.DecodeString(encodedURL)
	originURL := string(b)
	require.Equal(t, fmt.Sprintf("http://127.0.0.1:%d/echo?a=1", dp.Port()), originURL)
	require.NotEmpty(t, u.Query().Get("nonce"))
	require.NotEmpty(t, u.Query().Get("code_challenge"))
	cookie := initialResp.Header.Get("Set-Cookie")
	require.Regexp(t, `^htnn_oidc_nonce_[^=]+=[^;]+; Max-Age=3600; HttpOnly$`, cookie)

	// request the authorization endpoint
	authRedirectURL = strings.Replace(authRedirectURL, "http://hydra:4444", "http://127.0.0.1:4444", 1)
	req, err := http.NewRequest("GET", authRedirectURL, nil)
	require.NoError(t, err)
	loginPageResp, err := client.Do(req)

	// redirect to /login
	loginPageURL := loginPageResp.Header.Get("Location")
	loginPageResp, err = client.Get(loginPageURL)
	require.NoError(t, err)
	require.Equal(t, 200, loginPageResp.StatusCode)

	// Extract CSRF and Challenge
	loginBodyBytes, err := io.ReadAll(loginPageResp.Body)
	require.NoError(t, err)
	loginPageResp.Body.Close()
	csrfToken, challenge, err := extractCSRFAndChallenge(loginBodyBytes)
	require.NoError(t, err)

	// Submit login form
	loginResp, err := submitLoginForm(client, loginPageURL, csrfToken, challenge)
	require.NoError(t, err)
	require.Equal(t, 302, loginResp.StatusCode)

	// redirect back to the authorization endpoint
	postLoginRedirectURL := loginResp.Header.Get("Location")
	postLoginRedirectURL = strings.Replace(postLoginRedirectURL, "http://hydra:4444", "http://127.0.0.1:4444", 1)
	postLoginResp, err := client.Get(postLoginRedirectURL)
	require.NoError(t, err)

	// Redirect to consent page
	consentPageURL := postLoginResp.Header.Get("Location")
	consentPageResp, err := client.Get(consentPageURL)
	require.NoError(t, err)

	// Extract CSRF and Challenge from the consent page;
	// Submit a consent page form
	consentBodyBytes, err := io.ReadAll(consentPageResp.Body)
	require.NoError(t, err)
	consentPageResp.Body.Close()
	csrfToken, challenge, err = extractCSRFAndChallenge(consentBodyBytes)
	require.NoError(t, err)
	consentResp, err := submitConsentForm(client, consentPageURL, csrfToken, challenge)
	require.NoError(t, err)
	require.Equal(t, 302, consentResp.StatusCode)

	// redirect back to the authorization endpoint
	postConsentRedirectURL := consentResp.Header.Get("Location")
	require.NotEmpty(t, postConsentRedirectURL)
	postConsentRedirectURL = strings.Replace(postConsentRedirectURL, "http://hydra:4444", "http://127.0.0.1:4444", 1)
	callbackResp, err := client.Get(postConsentRedirectURL)
	require.NoError(t, err)
	require.Equal(t, 303, callbackResp.StatusCode)

	// Follow the redirect back to the application (callback URL)
	callbackRedirectURL := callbackResp.Header.Get("Location")
	applicationCallbackResp, err := client.Get(callbackRedirectURL)
	require.NoError(t, err)
	require.Equal(t, 302, applicationCallbackResp.StatusCode)

	// verify htnn_oidc_auth_data cookie
	cookies := applicationCallbackResp.Header["Set-Cookie"]
	require.NotEmpty(t, cookies)
	cookieFound := false
	var name, value string
	for _, cookie := range cookies {
		if strings.Contains(cookie, "htnn_oidc_auth_data") {
			parts := strings.SplitN(strings.Split(cookie, ";")[0], "=", 2)
			name = parts[0]
			value = parts[1]
			cookieFound = true
			break
		}
	}
	require.True(t, cookieFound)

	cookieDecoder := securecookie.New([]byte(hydra.ClientSecret), []byte(cookieEncryptionKey))
	authData := &oidc.AuthData{}
	err = cookieDecoder.Decode(name, value, authData)
	require.NoError(t, err)
	require.NotEmpty(t, authData.UserInfoJSON)

	var userInfo map[string]interface{}
	err = json.Unmarshal([]byte(authData.UserInfoJSON), &userInfo)
	require.NoError(t, err, "userinfo json should be valid")

	sub, ok := userInfo["sub"]
	require.True(t, ok, "userinfo json should contain 'sub' field")
	require.NotEmpty(t, sub, "'sub' field should not be empty")
}

func submitLoginForm(client *http.Client, loginURL, csrfToken, challenge string) (*http.Response, error) {
	form := url.Values{}
	form.Set("email", "foo@bar.com")
	form.Set("password", "foobar")
	form.Set("_csrf", csrfToken)
	form.Set("challenge", challenge)

	loginReq, err := http.NewRequest("POST", loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	// important headers
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Referer", loginURL)

	resp, err := client.Do(loginReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func submitConsentForm(client *http.Client, consentURL, csrfToken, challenge string) (*http.Response, error) {
	// Fill out the form
	form := url.Values{}
	form.Set("_csrf", csrfToken)
	form.Set("challenge", challenge)
	form.Add("grant_scope", "openid")
	form.Add("grant_scope", "offline_access")
	form.Set("submit", "Allow access")

	consentReq, err := http.NewRequest("POST", consentURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	// important headers
	consentReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	consentReq.Header.Set("Referer", consentURL)

	resp, err := client.Do(consentReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func extractCSRFAndChallenge(body []byte) (csrfToken, challenge string, err error) {
	bodyStr := string(body)

	reCsrf := regexp.MustCompile(`name=["']_csrf["']\s+value=["']([^"']+)["']`)
	reChallenge := regexp.MustCompile(`name=["']challenge["']\s+value=["']([^"']+)["']`)

	csrfMatches := reCsrf.FindStringSubmatch(bodyStr)
	challengeMatches := reChallenge.FindStringSubmatch(bodyStr)

	if len(csrfMatches) < 2 || len(challengeMatches) < 2 {
		preview := bodyStr
		if len(bodyStr) > 500 {
			preview = bodyStr[:500] + "..."
		}
		return "", "", fmt.Errorf("failed to extract csrf or challenge from body: %s", preview)
	}

	return csrfMatches[1], challengeMatches[1], nil
}
