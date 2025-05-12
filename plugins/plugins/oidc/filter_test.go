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

package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/securecookie"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"mosn.io/htnn/api/pkg/filtermanager/api"
	"mosn.io/htnn/api/plugins/tests/pkg/envoy"
	oidctype "mosn.io/htnn/types/plugins/oidc"
)

func getCfg() *config {
	return &config{
		CustomConfig: oidctype.CustomConfig{
			Config: oidctype.Config{
				ClientId:              "9119df09-b20b-4c08-ba08-72472dda2cd2",
				ClientSecret:          "dSYo5hBwjX_DC57_tfZHlfrDel",
				RedirectUrl:           "http://127.0.0.1:10000",
				IdTokenHeader:         "my-id-token",
				EnableUserinfoSupport: true,
				CookieEncryptionKey:   "1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p",
			},
		},
		oauth2Config:   &oauth2.Config{},
		verifier:       &oidc.IDTokenVerifier{},
		oidcProvider:   &oidc.Provider{},
		cookieEncoding: securecookie.New([]byte("dSYo5hBwjX_DC57_tfZHlfrDel"), []byte("1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p")),
		cookieEntryID:  "id",
	}
}

const userInfoClaimsJSON = `{"sub": "1234567890","profile": "https://example.com/user/profile","email": "user@example.com","email_verified": true}`

func getUserinfo() *oidc.UserInfo {
	o := &oidc.UserInfo{
		Subject:       "1234567890",
		Profile:       "https://example.com/profile",
		Email:         "user@example.com",
		EmailVerified: true,
	}
	val := reflect.ValueOf(o).Elem()
	field := val.FieldByName("claims")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().SetBytes([]byte(userInfoClaimsJSON))
	return o
}

func TestInitRequest(t *testing.T) {
	conf := getCfg()
	url := "http://host.docker.internal:4444/oauth2/auth?client_id=ef34cf65-016c-4b17-9864-8bd04dc22555&code_challenge=i3aZkytxb-6b4zvopxeT8AY21kon7EnJ7TlumdMlVuU&code_challenge_method=S256&nonce=yFyviTyEYAw&redirect_uri=http%3A%2F%2F127.0.0.1%3A10000%2Fecho&response_type=code&scope=openid&state=hqV183kqqtJxk_10F_5Y9"
	patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "AuthCodeURL", url)
	defer patches.Reset()

	cb := envoy.NewFilterCallbackHandler()
	f := factory(getCfg(), cb).(*filter)
	h := http.Header{}
	hdr := envoy.NewRequestHeaderMap(h)
	res := f.DecodeHeaders(hdr, true)
	resp := res.(*api.LocalResponse)
	assert.Equal(t, url, resp.Header.Get("Location"))
	// other fields are checked in the integration test
}

func TestCallback(t *testing.T) {
	conf := getCfg()
	conf.DisableAccessTokenRefresh = true

	verifier := oauth2.GenerateVerifier()
	state := generateState(verifier, conf.ClientSecret, "https://127.0.0.1:2379/x?y=1")
	rawIDToken := "rawIDToken"
	accessToken := "accessToken"
	token := (&oauth2.Token{
		AccessToken: accessToken,
	}).WithExtra(map[string]interface{}{
		"id_token": rawIDToken,
	})
	nonce, _ := conf.cookieEncoding.Encode("htnn_oidc_nonce_id", "xxx")

	tests := []struct {
		name                    string
		state                   string
		cookie                  string
		mock                    func() *gomonkey.Patches
		res                     api.ResultAction
		checkRedirectClientBack func(f *filter, headers http.Header)
	}{
		{
			name:   "sanity",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{
					Nonce: "xxx", Expiry: time.Now().Add(2 * time.Hour),
				}, nil)
				patches.ApplyMethodReturn(conf.oidcProvider, "UserInfo", getUserinfo(), nil)
				return patches
			},
			checkRedirectClientBack: func(f *filter, headers http.Header) {
				s := headers.Get("Location")
				assert.Equal(t, "https://127.0.0.1:2379/x?y=1", s)
				cookie := headers.Get("Set-Cookie")
				assert.Contains(t, cookie, "Max-Age=7199;")

				// verify the cookie value
				v := strings.SplitN(strings.Split(cookie, ";")[0], "=", 2)[1]
				h := http.Header{}
				hdr := envoy.NewRequestHeaderMap(h)
				assert.Equal(t, api.Continue, f.attachInfo(hdr, v))

				authData, ok := hdr.Get(conf.UserinfoHeader)
				assert.True(t, ok, "Userinfo header should exist")
				assert.NotEmpty(t, authData, "Userinfo header should not be empty")
				data, err := base64.RawURLEncoding.DecodeString(authData)
				assert.Nil(t, err, "base64 decode failed")

				var js json.RawMessage
				assert.Nil(t, json.Unmarshal(data, &js))
				assert.JSONEq(t, userInfoClaimsJSON, string(js))
				bearer, _ := hdr.Get("authorization")
				assert.Equal(t, "Bearer accessToken", bearer)
			},
		},
		{
			name:   "userinfo support disabled",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{
					Nonce: "xxx", Expiry: time.Now().Add(2 * time.Hour),
				}, nil)
				conf.EnableUserinfoSupport = false
				return patches
			},
			checkRedirectClientBack: func(f *filter, headers http.Header) {
				s := headers.Get("Location")
				assert.Equal(t, "https://127.0.0.1:2379/x?y=1", s)
				cookies := headers.Get("Set-Cookie")
				assert.NotEmpty(t, cookies)
				assert.Contains(t, cookies, "Max-Age=7199;")

				// verify the cookies value
				v := strings.SplitN(strings.Split(cookies, ";")[0], "=", 2)[1]
				k := strings.SplitN(strings.Split(cookies, ";")[0], "=", 2)[0]

				authData := &AuthData{}
				cookieDecoder := securecookie.New([]byte(conf.ClientSecret), []byte(conf.CookieEncryptionKey))
				err := cookieDecoder.Decode(k, v, authData)
				assert.NoError(t, err)
				assert.Empty(t, authData.UserInfoJSON)
				conf.EnableUserinfoSupport = true
			},
		},
		{
			name:   "fetch userinfo endpoint failed",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{
					Nonce: "xxx", Expiry: time.Now().Add(2 * time.Hour),
				}, nil)
				patches.ApplyMethodReturn(conf.oidcProvider, "UserInfo", nil, errors.New("test userinfo error"))
				return patches
			},
			res: &api.LocalResponse{Code: 403, Msg: "failed to fetch userinfo"},
		},
		{
			name:   "ttl with access token expiry",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				token := (&oauth2.Token{
					Expiry:      time.Now().Add(2 * time.Minute),
					AccessToken: accessToken,
				}).WithExtra(map[string]interface{}{
					"id_token": rawIDToken,
				})
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{
					Nonce: "xxx", Expiry: time.Now().Add(2 * time.Hour),
				}, nil)
				patches.ApplyMethodReturn(conf.oidcProvider, "UserInfo", getUserinfo(), nil)
				return patches
			},
			checkRedirectClientBack: func(f *filter, headers http.Header) {
				s := headers.Get("Location")
				assert.Equal(t, "https://127.0.0.1:2379/x?y=1", s)
				cookie := headers.Get("Set-Cookie")
				assert.Contains(t, cookie, "Max-Age=119;")
			},
		},
		{
			name:   "bad state",
			state:  state + "x",
			cookie: "htnn_oidc_nonce_id=" + nonce,
			res:    &api.LocalResponse{Code: 403, Msg: "bad state"},
		},
		{
			name:   "failed to exchange",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", nil, errors.New("timed out"))
				return patches
			},
			res: &api.LocalResponse{Code: 503, Msg: "failed to exchange code to the token"},
		},
		{
			name:   "failed to lookup token",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", &oauth2.Token{}, nil)
				return patches
			},
			res: &api.LocalResponse{Code: 503, Msg: "failed to lookup id token"},
		},
		{
			name:   "bad token",
			state:  state,
			cookie: "htnn_oidc_nonce_id=" + nonce,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", nil, errors.New("ouch"))
				return patches
			},
			res: &api.LocalResponse{Code: 503, Msg: "bad token"},
		},
		{
			name:   "bad nonce",
			state:  state,
			cookie: "htnn_oidc_nonce_id=xxy",
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{Nonce: "xxx"}, nil)
				return patches
			},
			res: &api.LocalResponse{Code: 403, Msg: "bad nonce"},
		},
		{
			name:  "bad nonce, no cookie",
			state: state,
			mock: func() *gomonkey.Patches {
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
				patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{Nonce: "xxx"}, nil)
				return patches
			},
			res: &api.LocalResponse{Code: 403, Msg: "bad nonce"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				patches := tt.mock()
				defer patches.Reset()
			}

			cb := envoy.NewFilterCallbackHandler()
			f := factory(conf, cb).(*filter)
			h := http.Header{}
			h.Set(":path", "/echo?code=123&state="+tt.state)
			h.Set("cookie", tt.cookie)
			hdr := envoy.NewRequestHeaderMap(h)
			res := f.DecodeHeaders(hdr, true)
			if tt.res != nil {
				assert.Equal(t, tt.res, res)
			}

			if tt.checkRedirectClientBack != nil {
				resp := res.(*api.LocalResponse)
				assert.Equal(t, http.StatusFound, resp.Code, resp.Msg)
				tt.checkRedirectClientBack(f, resp.Header)
			}
		})
	}
}

func TestBadOIDCTokenCookie(t *testing.T) {
	cb := envoy.NewFilterCallbackHandler()
	f := factory(getCfg(), cb).(*filter)
	h := http.Header{}
	h.Set("Cookie", "htnn_oidc_auth_data_id=xxx")
	hdr := envoy.NewRequestHeaderMap(h)
	res := f.DecodeHeaders(hdr, true)
	resp := res.(*api.LocalResponse)
	assert.Equal(t, 403, resp.Code)
	assert.Equal(t, "bad oidc cookie", resp.Msg)
}

type mockTokenSource struct {
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return nil, nil
}

// Note: If the patch does not work, try using -gcflags=all=-l to disable inlining optimizations.
func TestAttachInfo(t *testing.T) {
	conf := getCfg()
	verifier := oauth2.GenerateVerifier()
	state := generateState(verifier, conf.ClientSecret, "https://127.0.0.1:2379/x?y=1")
	rawIDToken := "rawIDToken"
	rawIDToken2 := "rawIDToken2"
	accessToken := "accessToken"
	accessToken2 := "accessToken2"
	refreshToken := "refreshToken"
	token := (&oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(1 * time.Hour),
	}).WithExtra(map[string]interface{}{
		"id_token": rawIDToken,
	})
	nonce, _ := conf.cookieEncoding.Encode("htnn_oidc_nonce_id", "xxx")

	patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "Exchange", token, nil)
	patches.ApplyMethodReturn(conf.verifier, "Verify", &oidc.IDToken{
		Nonce: "xxx", Expiry: time.Now().Add(2 * time.Hour),
	}, nil)
	patches.ApplyMethodReturn(conf.oidcProvider, "UserInfo", getUserinfo(), nil)
	defer patches.Reset()

	cb := envoy.NewFilterCallbackHandler()
	f := factory(conf, cb).(*filter)
	h := http.Header{}
	h.Set(":path", "/echo?code=123&state="+state)
	h.Set("cookie", "htnn_oidc_nonce_id="+nonce)
	hdr := envoy.NewRequestHeaderMap(h)
	res := f.DecodeHeaders(hdr, true)
	resp := res.(*api.LocalResponse)
	cookie := resp.Header.Get("Set-Cookie")
	assert.Contains(t, cookie, "Max-Age=7199;") // the ttl is from id token expiry

	v := strings.SplitN(strings.Split(cookie, ";")[0], "=", 2)[1]

	expiredToken, _ := conf.cookieEncoding.Encode("htnn_oidc_auth_data_id", AuthData{
		Oauth2Token: &oauth2.Token{
			AccessToken:  "expiredToken",
			Expiry:       time.Now().Add(-1 * time.Hour),
			RefreshToken: refreshToken,
		},
	})
	expiredAndUnrefreshableToken, _ := conf.cookieEncoding.Encode("htnn_oidc_auth_data_id", AuthData{
		Oauth2Token: &oauth2.Token{
			AccessToken: "expiredToken",
			Expiry:      time.Now().Add(-1 * time.Hour),
		},
	})

	refreshedAccessToken := (&oauth2.Token{
		AccessToken: accessToken2,
		Expiry:      time.Now().Add(1 * time.Hour),
	}).WithExtra(map[string]interface{}{
		"id_token": rawIDToken2,
	})

	tests := []struct {
		name          string
		encodedToken  string
		mock          func() *gomonkey.Patches
		config        *config
		res           api.ResultAction
		authorization string
		idTokenSet    string
		checkCookie   func(cookie string)
		checkHeader   func(hdr api.RequestHeaderMap)
	}{
		{
			name:         "sanity",
			encodedToken: v,
			mock: func() *gomonkey.Patches {
				tkSrc := &mockTokenSource{}
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "TokenSource", tkSrc)
				return patches
			},
			res:           api.Continue,
			authorization: "Bearer accessToken",
			idTokenSet:    rawIDToken,
			checkHeader: func(hdr api.RequestHeaderMap) {
				userInfoHeader, _ := hdr.Get(conf.UserinfoHeader)
				assert.NotEmpty(t, userInfoHeader)
				userinfoJSON, _ := base64.RawURLEncoding.DecodeString(userInfoHeader)
				var raw json.RawMessage
				err := getUserinfo().Claims(&raw)
				assert.NoError(t, err)
				assert.Equal(t, raw, json.RawMessage(userinfoJSON))
			},
		},
		{
			name:         "refresh token",
			encodedToken: expiredToken,
			mock: func() *gomonkey.Patches {
				tkSrc := &mockTokenSource{}
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "TokenSource", tkSrc)
				patches.ApplyMethodReturn(tkSrc, "Token", refreshedAccessToken, nil)
				return patches
			},
			res:           api.Continue,
			authorization: "Bearer accessToken2",
			idTokenSet:    rawIDToken2,
			checkCookie: func(cookie string) {
				// A new cookie should be set
				assert.Contains(t, cookie, "Max-Age=3599;")

				v := strings.SplitN(strings.Split(cookie, ";")[0], "=", 2)[1]
				var decoded AuthData
				err := conf.cookieEncoding.Decode(f.CookieName("auth_data"), v, &decoded)
				assert.NoError(t, err, "failed to decode cookie")
				assert.NotEmpty(t, decoded.UserInfoJSON, "UserInfoJSON should not be empty")
			},
			checkHeader: func(hdr api.RequestHeaderMap) {
				userInfoHeader, _ := hdr.Get(conf.UserinfoHeader)
				assert.NotEmpty(t, userInfoHeader)
				userinfoJSON, _ := base64.RawURLEncoding.DecodeString(userInfoHeader)
				var raw json.RawMessage
				err := getUserinfo().Claims(&raw)
				assert.NoError(t, err)
				assert.Equal(t, raw, json.RawMessage(userinfoJSON))
			},
		},
		{
			name:         "unrefreshable token",
			encodedToken: expiredAndUnrefreshableToken,
			res:          &api.LocalResponse{Code: 401},
		},
		{
			name:         "failed to refresh token",
			encodedToken: expiredToken,
			mock: func() *gomonkey.Patches {
				tkSrc := &mockTokenSource{}
				patches := gomonkey.ApplyMethodReturn(conf.oauth2Config, "TokenSource", tkSrc)
				patches.ApplyMethodReturn(tkSrc, "Token", nil, errors.New("failed to refresh"))
				return patches
			},
			res: &api.LocalResponse{Code: 401},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				patches := tt.mock()
				defer patches.Reset()
			}

			conf := getCfg()
			if tt.config != nil {
				conf = tt.config
			}
			f := factory(conf, cb).(*filter)
			h = http.Header{}
			hdr = envoy.NewRequestHeaderMap(h)

			assert.Equal(t, tt.res, f.attachInfo(hdr, tt.encodedToken))
			bearer, _ := hdr.Get("authorization")
			idTokenSet, _ := hdr.Get("my-id-token")
			assert.Equal(t, tt.authorization, bearer)
			assert.Equal(t, tt.idTokenSet, idTokenSet)

			if tt.checkHeader != nil {
				tt.checkHeader(hdr)
			}
			if tt.checkCookie != nil {
				h = http.Header{}
				hdr := envoy.NewResponseHeaderMap(h)
				f.EncodeHeaders(hdr, true)
				cookie, _ := hdr.Get("Set-Cookie")
				tt.checkCookie(cookie)
			}
		})
	}
}

func TestFetchUserInfoIfEnabled(t *testing.T) {
	conf := getCfg()
	f := &filter{config: conf}
	token := &oauth2.Token{AccessToken: "dummy"}
	ctx := context.Background()

	patches := gomonkey.ApplyMethodReturn(conf.oidcProvider, "UserInfo", getUserinfo(), nil)
	defer patches.Reset()

	t.Run("userinfoFormat raw json", func(t *testing.T) {
		conf.UserinfoFormat = oidctype.UserinfoFormatEnums_RAW_JSON

		result, err := f.fetchUserInfoIfEnabled(ctx, token)
		assert.NoError(t, err)
		assert.JSONEq(t, userInfoClaimsJSON, result)
	})

	t.Run("userinfoFormat base64", func(t *testing.T) {
		conf.UserinfoFormat = oidctype.UserinfoFormatEnums_BASE64
		result, err := f.fetchUserInfoIfEnabled(ctx, token)
		assert.NoError(t, err)

		data, err := base64.StdEncoding.DecodeString(result)
		assert.NoError(t, err)
		assert.JSONEq(t, userInfoClaimsJSON, string(data))
	})

	t.Run("userinfoFormat base64url", func(t *testing.T) {
		conf.UserinfoFormat = oidctype.UserinfoFormatEnums_BASE64URL
		result, err := f.fetchUserInfoIfEnabled(ctx, token)
		assert.NoError(t, err)

		data, err := base64.RawURLEncoding.DecodeString(result)
		assert.NoError(t, err)
		assert.JSONEq(t, userInfoClaimsJSON, string(data))
	})

	t.Run("userinfo support disabled", func(t *testing.T) {
		conf.EnableUserinfoSupport = false
		result, err := f.fetchUserInfoIfEnabled(ctx, token)
		assert.NoError(t, err)
		assert.Equal(t, "", result)
		conf.EnableUserinfoSupport = true // 还原
	})

	t.Run("userinfo fetch failed", func(t *testing.T) {
		patches.Reset()
		patches = gomonkey.ApplyMethodReturn(conf.oidcProvider, "UserInfo", nil, errors.New("userinfo err"))
		defer patches.Reset()

		_, err := f.fetchUserInfoIfEnabled(ctx, token)
		assert.Error(t, err)
	})
}
