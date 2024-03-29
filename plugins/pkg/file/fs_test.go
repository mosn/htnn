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
