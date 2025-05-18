package basicauth

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "google.golang.org/protobuf/encoding/protojson"

    "mosn.io/htnn/types/plugins/basicauth"
)

func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name  string
        input string
        err   string
    }{
        {
            name:  "empty credentials",
            input: `{"credentials":{}}`,
            err:   "at least one username-password pair must be specified",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            conf := &basicauth.Config{}
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