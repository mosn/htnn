package ext_auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConfTimeout(t *testing.T) {
	s := `{"http_service":{
		"timeout": "10s"
	}}`
	conf := &config{}
	protojson.Unmarshal([]byte(s), conf)
	conf.Init(nil)
	assert.Equal(t, 10*time.Second, conf.client.Timeout)
}

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
			name:  "invalid HttpService.Url",
			input: `{"http_service":{"url":"127.0.0.1"}}`,
			err:   "invalid HttpService.Url: value must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &config{}
			protojson.Unmarshal([]byte(tt.input), conf)
			err := conf.Validate()
			assert.NotNil(t, err)
			assert.ErrorContains(t, err, tt.err)
		})
	}
}
