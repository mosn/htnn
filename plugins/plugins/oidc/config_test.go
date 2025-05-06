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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	"mosn.io/htnn/types/plugins/oidc"
)

func newProtoConfigForTest(protoConfigOpt func(*oidc.Config)) *config {
	cfg := &config{}
	protoConfigOpt(&cfg.CustomConfig.Config)
	return cfg
}

func TestBadIssuer(t *testing.T) {
	c := newProtoConfigForTest(func(cfg *oidc.Config) {
		cfg.Issuer = "http://1.1.1.1"
		cfg.Timeout = &durationpb.Duration{Seconds: 1}
	})
	err := c.Init(nil)
	assert.Error(t, err)
}

func TestDefaultValue(t *testing.T) {
	c := newProtoConfigForTest(func(cfg *oidc.Config) {
		cfg.Issuer = "http://1.1.1.1"
		cfg.ClientId = "test-client"
		cfg.ClientSecret = "test-secret"
		cfg.RedirectUrl = "http://localhost/callback"
		cfg.EnableUserinfoSupport = true
		cfg.CookieEncryptionKey = "1234567890123456"
		cfg.Timeout = &durationpb.Duration{Seconds: 1}
	})
	c.Init(nil)
	assert.Equal(t, "x-id-token", c.IdTokenHeader)
	assert.Equal(t, "x-userinfo", c.UserinfoHeader)
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "bad issuer url",
			input: `{"clientId":"a", "clientSecret":"b", "issuer":"google.com"}`,
			err:   "invalid Config.Issuer:",
		},
		{
			name:  "leeway can be 0s",
			input: `{"clientId":"a", "clientSecret":"b", "issuer":"https://google.com", "redirectUrl":"http://127.0.0.1:10000/echo", "accessTokenRefreshLeeway":"0s"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config{}
			err := protojson.Unmarshal([]byte(tt.input), conf)
			if err == nil {
				err = conf.Validate()
			}
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}

func TestCtxWithClient(t *testing.T) {
	// Test configuration
	conf := &config{
		opTimeout: 30 * time.Second,
	}

	t.Run("should inject new client when no HTTPClient exists", func(t *testing.T) {
		ctx := context.Background()

		resultCtx := conf.ctxWithClient(ctx)

		client, ok := resultCtx.Value(oauth2.HTTPClient).(*http.Client)
		if !ok {
			t.Fatal("Expected HTTPClient in context")
		}

		if client.Timeout != conf.opTimeout {
			t.Errorf("Expected timeout %v, got %v", conf.opTimeout, client.Timeout)
		}
	})

	t.Run("should preserve existing HTTPClient when present", func(t *testing.T) {
		existingClient := &http.Client{Timeout: 10 * time.Second}
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, existingClient)

		resultCtx := conf.ctxWithClient(ctx)

		retrievedClient := resultCtx.Value(oauth2.HTTPClient).(*http.Client)
		if retrievedClient != existingClient {
			t.Error("Should not replace existing HTTPClient")
		}
	})

	t.Run("should skip injection for non-*http.Client values", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, "invalid-value")

		resultCtx := conf.ctxWithClient(ctx)

		val := resultCtx.Value(oauth2.HTTPClient)
		if _, ok := val.(*http.Client); ok {
			t.Error("Should not override non-client values")
		}
		if val != "invalid-value" {
			t.Error("Should preserve original invalid value")
		}
	})
}

func TestUserinfo(t *testing.T) {
	// without an encryption key
	conf1 := newProtoConfigForTest(func(cfg *oidc.Config) {
		cfg.Issuer = "http://1.1.1.1"
		cfg.ClientId = "test-client"
		cfg.ClientSecret = "test-secret"
		cfg.RedirectUrl = "http://localhost/callback"
		cfg.EnableUserinfoSupport = true
		cfg.Timeout = &durationpb.Duration{Seconds: 1}
	})
	conf1.Init(nil)
	err := conf1.Validate()
	assert.ErrorContains(t, err, "value length must be 16, 24 or 32 bytes")

	// with an invalid length encryption key
	conf2 := newProtoConfigForTest(func(cfg *oidc.Config) {
		cfg.Issuer = "http://1.1.1.1"
		cfg.ClientId = "test-client"
		cfg.ClientSecret = "test-secret"
		cfg.RedirectUrl = "http://localhost/callback"
		cfg.EnableUserinfoSupport = true
		cfg.CookieEncryptionKey = "123"
		cfg.Timeout = &durationpb.Duration{Seconds: 1}
	})

	conf2.Init(nil)
	err = conf2.Validate()
	assert.IsType(t, oidc.ConfigValidationError{}, err)

	// with a valid length encryption key
	validKeys := []string{
		"1234567890123456",                 // AES-128
		"123456789012345678901234",         // AES-192
		"12345678901234567890123456789012", // AES-256
	}

	for _, key := range validKeys {
		conf3 := newProtoConfigForTest(func(cfg *oidc.Config) {
			cfg.Issuer = "http://1.1.1.1"
			cfg.ClientId = "test-client"
			cfg.ClientSecret = "test-secret"
			cfg.RedirectUrl = "http://localhost/callback"
			cfg.EnableUserinfoSupport = true
			cfg.CookieEncryptionKey = key
			cfg.Timeout = &durationpb.Duration{Seconds: 1}
		})
		err = conf3.Init(nil)
		if err != nil {
			assert.NotContains(t, err.Error(), "block key")
		}
		err = conf3.Validate()
		assert.Nil(t, err)
	}
}
