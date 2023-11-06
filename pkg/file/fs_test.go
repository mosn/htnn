package file

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileMtimeDetection(t *testing.T) {
	defaultFs = newFS(2000 * time.Millisecond)

	tmpfile, _ := os.CreateTemp("", "example")
	defer os.Remove(tmpfile.Name()) // clean up

	f, err := Stat(tmpfile.Name())
	assert.Nil(t, err)
	assert.False(t, IsChanged(f))
	time.Sleep(1000 * time.Millisecond)
	tmpfile.Write([]byte("bls"))
	tmpfile.Close()
	assert.False(t, IsChanged(f))

	time.Sleep(2500 * time.Millisecond)
	assert.True(t, IsChanged(f))
	assert.True(t, Update(f))
	assert.False(t, IsChanged(f))
}
