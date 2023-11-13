package plugins

import "os"

func writeTempFile(s string) *os.File {
	tmpfile, _ := os.CreateTemp("", "test")
	tmpfile.Write([]byte(s))
	return tmpfile
}
