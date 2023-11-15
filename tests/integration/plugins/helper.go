package plugins

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeTempFile(s string) *os.File {
	tmpfile, _ := os.CreateTemp("", "test")
	tmpfile.Write([]byte(s))
	return tmpfile
}

func waitServiceUp(t *testing.T, port string, msg string) {
	require.Eventually(t, func() bool {
		c, err := net.DialTimeout("tcp", port, 10*time.Millisecond)
		if err != nil {
			return false
		}
		c.Close()
		return true
	}, 10*time.Second, 50*time.Millisecond, msg)
}
