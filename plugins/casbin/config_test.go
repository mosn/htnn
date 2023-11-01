package casbin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "no required fields",
			input: `{}`,
		},
		{
			name: "empty policy",
			input: `{
				"rule": {
					"model": "./config/model.conf",
					"policy": ""
				},
				"token": {
					"source": "header",
					"name": "role"
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&parser{}).Validate([]byte(tt.input))
			assert.NotNil(t, err)
		})
	}
}
