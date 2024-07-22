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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"mosn.io/htnn/api/pkg/filtermanager/api"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks   api.FilterCallbackHandler
	config      *config
	tokenCookie *http.Cookie
}

type Tokens struct {
	IDToken     string        `json:"id_token"`
	Oauth2Token *oauth2.Token `json:"oauth_token"`
}

func generateState(verifier string, secret string, url string) string {
	encodedRedirectURL := base64.URLEncoding.EncodeToString([]byte(url))
	state := fmt.Sprintf("%s.%s", verifier, encodedRedirectURL)
	signature := signState(state, secret)
	// fmt: verifier.originURL.signature
	return fmt.Sprintf("%s.%s", state, signature)
}

func verifyState(state string, secret string) bool {
	pieces := strings.Split(state, ".")
	if len(pieces) != 3 {
		return false
	}
	data := fmt.Sprintf("%s.%s", pieces[0], pieces[1])
	signature := signState(data, secret)
	return pieces[2] == signature
}

func signState(state string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(state))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (f *filter) CookieName(key string) string {
	return fmt.Sprintf("htnn_oidc_%s_%s", key, f.config.cookieEntryID)
}

func (f *filter) handleInitRequest(headers api.RequestHeaderMap) api.ResultAction {
	config := f.config
	o2conf := config.oauth2Config

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	nonce := base64.RawURLEncoding.EncodeToString(b)
	verifier := oauth2.GenerateVerifier()
	originURL := fmt.Sprintf("%s://%s%s", headers.Scheme(), headers.Host(), headers.Path())
	s := generateState(verifier, config.ClientSecret, originURL)
	url := o2conf.AuthCodeURL(s,
		// use PKCE to protect against CSRF attacks if possible
		// https://www.ietf.org/archive/id/draft-ietf-oauth-security-topics-22.html#name-countermeasures-6
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce))

	cookieName := f.CookieName("nonce")
	n, err := config.cookieEncoding.Encode(cookieName, nonce)
	if err != nil {
		api.LogErrorf("failed to encode cookie: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to encode cookie"}
	}
	cookieNonce := &http.Cookie{
		Name:     cookieName,
		Value:    n,
		MaxAge:   int(time.Hour.Seconds()),
		HttpOnly: true,
		// TODO: allow configuring the cookie attributes
	}

	return &api.LocalResponse{
		Code: http.StatusFound,
		Header: http.Header{
			"Location":   []string{url},
			"Set-Cookie": []string{cookieNonce.String()},
		},
	}
}

func (f *filter) calculateTokenTTL(accessTokenExpiry time.Time, idTokenExpiry time.Time, refreshEnabled bool) int {
	if refreshEnabled {
		// As the access token refresh is enabled, we only need to consider the expiry of id token
		return int(time.Until(idTokenExpiry).Seconds())
	}

	// Use the min expiry between id token and access token as the expiry
	if accessTokenExpiry.IsZero() {
		// According to https://openid.net/specs/openid-connect-core-1_0.html#IDToken,
		// the expiry of id token is required.
		// Meanwhile, the expiry of access token is optional.
		return int(time.Until(idTokenExpiry).Seconds())
	}
	return int(min(
		time.Until(accessTokenExpiry).Seconds(),
		time.Until(idTokenExpiry).Seconds()))
}

func getIDToken(token *oauth2.Token) (string, bool) {
	rawIDToken, ok := token.Extra("id_token").(string)
	return rawIDToken, ok
}

func (f *filter) refreshEnabled(token *oauth2.Token) bool {
	return !f.config.DisableAccessTokenRefresh && token.RefreshToken != ""
}

func (f *filter) handleCallback(headers api.RequestHeaderMap, query url.Values) api.ResultAction {
	config := f.config
	o2conf := config.oauth2Config
	ctx := context.Background()
	code := query.Get("code")
	state := query.Get("state")

	// Here we provide the mechanism below to ensure the id token is client's:
	// 1. sign the state to avoid being forged by the attacker
	// 2. use PKCE to ensure the code is bound with the state, which is trusted after being verified
	// 3. use nonce to ensure the id token is coming from the authorization request we initiated
	if !verifyState(state, config.ClientSecret) {
		api.LogInfof("bad state: %s", state)
		return &api.LocalResponse{Code: 403, Msg: "bad state"}
	}
	verifier, encodedURL, _ := strings.Cut(state, ".")
	b, _ := base64.URLEncoding.DecodeString(encodedURL)
	originURL := string(b)

	ctx = config.ctxWithClient(ctx)
	oauth2Token, err := o2conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		api.LogErrorf("failed to exchange code to the token: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to exchange code to the token"}
	}

	rawIDToken, ok := getIDToken(oauth2Token)
	if !ok {
		api.LogErrorf("failed to lookup id token: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to lookup id token"}
	}

	idToken, err := config.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		api.LogInfof("bad token: %s", err)
		return &api.LocalResponse{Code: 503, Msg: "bad token"}
	}

	if !config.SkipNonceVerify {
		cookieName := f.CookieName("nonce")
		nonce := headers.Cookie(cookieName)
		if nonce == nil {
			api.LogInfof("bad nonce, expected %s", idToken.Nonce)
			return &api.LocalResponse{Code: 403, Msg: "bad nonce"}
		}

		var p string
		err := config.cookieEncoding.Decode(cookieName, nonce.Value, &p)
		if err != nil || p != idToken.Nonce {
			if err != nil {
				api.LogInfof("bad nonce: %s, expected %s", err, idToken.Nonce)
			} else {
				api.LogInfof("bad nonce: %s, expected %s", p, idToken.Nonce)
			}
			return &api.LocalResponse{Code: 403, Msg: "bad nonce"}
		}
	}

	cookie, err := f.saveTokenAsCookie(ctx, oauth2Token, rawIDToken)
	if err != nil {
		return &api.LocalResponse{Code: 503, Msg: "failed to save token"}
	}

	return &api.LocalResponse{
		Code: http.StatusFound,
		Header: http.Header{
			"Location":   []string{originURL},
			"Set-Cookie": []string{cookie.String()},
		},
	}
}

func (f *filter) attachInfo(headers api.RequestHeaderMap, encodedToken string) api.ResultAction {
	config := f.config
	ctx := context.Background()

	tokens := &Tokens{}
	cookieName := f.CookieName("token")
	err := config.cookieEncoding.Decode(cookieName, encodedToken, tokens)
	if err != nil {
		api.LogInfof("bad oidc cookie: %s, err: %v", encodedToken, err)
		return &api.LocalResponse{Code: 403, Msg: "bad oidc cookie"}
	}

	oauth2Token := tokens.Oauth2Token
	rawIDToken := tokens.IDToken
	if f.refreshEnabled(oauth2Token) {
		tokenSrc := config.oauth2Config.TokenSource(context.Background(), oauth2Token)
		tokenSrc = oauth2.ReuseTokenSourceWithExpiry(oauth2Token, tokenSrc, config.refreshLeeway)
		possibleRefreshedToken, err := tokenSrc.Token()
		if err != nil {
			api.LogWarnf("failed to refresh access token %s, err: %v, refresh token: %s",
				oauth2Token.AccessToken, err, oauth2Token.RefreshToken)
			return &api.LocalResponse{Code: 401}
		}

		if possibleRefreshedToken.AccessToken != oauth2Token.AccessToken {
			// token refreshed
			oauth2Token = possibleRefreshedToken
			newIDToken, ok := getIDToken(oauth2Token)
			if ok {
				rawIDToken = newIDToken
			}

			f.tokenCookie, err = f.saveTokenAsCookie(ctx, possibleRefreshedToken, rawIDToken)
			if err != nil {
				return &api.LocalResponse{Code: 503, Msg: "failed to save token"}
			}
		}

	} else {
		ok := oauth2Token.Valid()
		if !ok {
			api.LogInfo("access token is not valid")
			return &api.LocalResponse{Code: 401}
		}
	}

	headers.Set("authorization", fmt.Sprintf("%s %s", oauth2Token.Type(), oauth2Token.AccessToken))
	headers.Set(config.IdTokenHeader, rawIDToken)
	return api.Continue
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	cookieName := f.CookieName("token")
	token := headers.Cookie(cookieName)
	if token != nil {
		return f.attachInfo(headers, token.Value)
	}

	query := headers.URL().Query()
	code := query.Get("code")
	if code == "" {
		return f.handleInitRequest(headers)
	}

	return f.handleCallback(headers, query)
}

func (f *filter) saveTokenAsCookie(ctx context.Context, oauth2Token *oauth2.Token, rawIDToken string) (*http.Cookie, error) {
	idToken, err := f.config.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		api.LogErrorf("bad token: %v", err)
		return nil, err
	}

	cookieName := f.CookieName("token")
	token, err := f.config.cookieEncoding.Encode(cookieName, Tokens{
		Oauth2Token: oauth2Token,
		IDToken:     rawIDToken,
	})
	if err != nil {
		api.LogErrorf("failed to encode cookie: %v", err)
		return nil, err
	}

	ttl := f.calculateTokenTTL(oauth2Token.Expiry, idToken.Expiry, f.refreshEnabled(oauth2Token))
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    token,
		MaxAge:   ttl,
		HttpOnly: true,
	}

	api.LogInfof("token saved as cookie %+v, client id: %s", cookie, f.config.ClientId)
	return cookie, nil
}

func (f *filter) EncodeHeaders(headers api.ResponseHeaderMap, endStream bool) api.ResultAction {
	if f.tokenCookie != nil {
		headers.Add("set-cookie", f.tokenCookie.String())
	}
	return api.Continue
}
