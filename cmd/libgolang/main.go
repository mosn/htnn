//go:build so

package main

import (
	_ "mosn.io/moe/plugins"
)

// Version is specified by build tag, in VERSION file
var (
	Version string = ""
)

func main() {}
