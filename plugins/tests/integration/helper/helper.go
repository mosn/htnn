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

package helper

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func WriteTempFile(s string) *os.File {
	tmpfile, _ := os.CreateTemp("", "test")
	tmpfile.Write([]byte(s))
	return tmpfile
}

func WaitServiceUp(t *testing.T, port string, service string) {
	msg := ""
	if service != "" {
		msg = fmt.Sprintf("Service is unavailble. Please run `docker-compose up %s` under ./plugins/tests/integration/testdata/services and ensure it is started", service)
	}
	require.Eventually(t, func() bool {
		c, err := net.DialTimeout("tcp", port, 10*time.Millisecond)
		if err != nil {
			return false
		}
		c.Close()
		return true
	}, 10*time.Second, 50*time.Millisecond, msg)
}
