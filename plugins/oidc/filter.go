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

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/request"
)

func factory(c interface{}, callbacks api.FilterCallbackHandler) api.Filter {
	return &filter{
		callbacks: callbacks,
		config:    c.(*config),
	}
}

type filter struct {
	api.PassThroughFilter

	callbacks api.FilterCallbackHandler
	config    *config
}

type Tokens struct {
	IDToken string `json:"id_token"`
}

func generateState(verifier string, secret string, url string) string {
	encodedRedirectUrl := base64.URLEncoding.EncodeToString([]byte(url))
	state := fmt.Sprintf("%s.%s", verifier, encodedRedirectUrl)
	signature := signState(state, secret)
	// fmt: verifier.originUrl.signature
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

func (f *filter) handleInitRequest(headers api.RequestHeaderMap) api.ResultAction {
	config := f.config
	o2conf := config.oauth2Config

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	nonce := base64.RawURLEncoding.EncodeToString(b)
	verifier := oauth2.GenerateVerifier()
	originUrl := fmt.Sprintf("%s://%s%s", headers.Scheme(), headers.Host(), headers.Path())
	s := generateState(verifier, config.ClientSecret, originUrl)
	url := o2conf.AuthCodeURL(s,
		// use PKCE to protect against CSRF attacks if possible
		// https://www.ietf.org/archive/id/draft-ietf-oauth-security-topics-22.html#name-countermeasures-6
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce))

	n, err := config.cookieEncoding.Encode("htnn_oidc_nonce", nonce)
	if err != nil {
		api.LogErrorf("failed to encode cookie: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to encode cookie"}
	}
	cookieNonce := &http.Cookie{
		Name:     "htnn_oidc_nonce",
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
	verifier, encodedUrl, _ := strings.Cut(state, ".")
	b, _ := base64.URLEncoding.DecodeString(encodedUrl)
	originUrl := string(b)

	ctx = ctxWithClient(ctx)
	oauth2Token, err := o2conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		api.LogErrorf("failed to exchange code to the token: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to exchange code to the token"}
	}

	// TODO: handle refresh_token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		api.LogErrorf("failed to lookup id token: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to lookup id token"}
	}

	idToken, err := config.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		api.LogInfof("bad token: %s", err)
		return &api.LocalResponse{Code: 403, Msg: "bad token"}
	}

	if !config.SkipNonceVerify {
		nonce, ok := request.GetCookies(headers)["htnn_oidc_nonce"]
		if !ok {
			api.LogInfof("bad nonce, expected %s", idToken.Nonce)
			return &api.LocalResponse{Code: 403, Msg: "bad nonce"}
		}

		var p string
		err := config.cookieEncoding.Decode("htnn_oidc_nonce", nonce.Value, &p)
		if err != nil || p != idToken.Nonce {
			if err != nil {
				api.LogInfof("bad nonce: %s, expected %s", err, idToken.Nonce)
			} else {
				api.LogInfof("bad nonce: %s, expected %s", p, idToken.Nonce)
			}
			return &api.LocalResponse{Code: 403, Msg: "bad nonce"}
		}
	}

	value := Tokens{
		IDToken: rawIDToken,
	}
	token, err := config.cookieEncoding.Encode("htnn_oidc_token", &value)
	if err != nil {
		api.LogErrorf("failed to encode cookie: %v", err)
		return &api.LocalResponse{Code: 503, Msg: "failed to encode cookie"}
	}

	cookie := &http.Cookie{
		Name:     "htnn_oidc_token",
		Value:    token,
		MaxAge:   int(time.Until(idToken.Expiry).Seconds()),
		HttpOnly: true,
	}
	return &api.LocalResponse{
		Code: http.StatusFound,
		Header: http.Header{
			"Location":   []string{originUrl},
			"Set-Cookie": []string{cookie.String()},
		},
	}
}

func (f *filter) attachInfo(headers api.RequestHeaderMap, encodedToken string) api.ResultAction {
	config := f.config

	value := Tokens{}
	err := config.cookieEncoding.Decode("htnn_oidc_token", encodedToken, &value)
	if err != nil {
		api.LogInfof("bad oidc cookie: %s, err: %s", encodedToken, err.Error())
		return &api.LocalResponse{Code: 403, Msg: "bad oidc cookie"}
	}
	headers.Set("authorization", fmt.Sprintf("Bearer %s", value.IDToken))
	return api.Continue
}

func (f *filter) DecodeHeaders(headers api.RequestHeaderMap, endStream bool) api.ResultAction {
	token, ok := request.GetCookies(headers)["htnn_oidc_token"]
	if ok {
		return f.attachInfo(headers, token.Value)
	}

	query := request.GetUrl(headers).Query()
	code := query.Get("code")
	if code == "" {
		return f.handleInitRequest(headers)
	}

	return f.handleCallback(headers, query)
}
