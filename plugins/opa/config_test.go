package opa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "at least one config type is required",
			input: `{}`,
			err:   "value is required",
		},
		{
			name: "empty url in remote",
			input: `{
				"remote": {
					"url": "",
					"policy": "authz"
				}
			}`,
			err: "invalid Remote.Url: value length must be at least 1 runes",
		},
		{
			name: "empty policy in remote",
			input: `{
				"remote": {
					"url": "http://127.0.0.1:8181",
					"policy": ""
				}
			}`,
			err: "invalid Remote.Policy: value length must be at least 1 runes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&parser{}).Validate([]byte(tt.input))
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}
