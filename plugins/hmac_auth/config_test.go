package hmac_auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConsumerConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name:  "name in signed_headers",
			input: `{"access_key":"a", "secret_key":"s", "signed_headers":[""]}`,
			err:   "value length must be at least 1 runes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConsumerConfig{}
			protojson.Unmarshal([]byte(tt.input), conf)
			err := conf.Validate()
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
