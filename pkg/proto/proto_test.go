package proto

import (
	"testing"

	xds "github.com/cncf/xds/go/xds/type/v3"
	"github.com/stretchr/testify/assert"
)

func TestMessageToAnyWithError(t *testing.T) {
	_, err := MessageToAnyWithError(nil)
	assert.NotNil(t, err)
}

func TestMessageToAny(t *testing.T) {
	out := MessageToAny(nil)
	assert.Nil(t, out)
	ts := xds.TypedStruct{}
	any1 := MessageToAny(&ts)
	assert.Equal(t, "type.googleapis.com/xds.type.v3.TypedStruct", any1.TypeUrl)
}
