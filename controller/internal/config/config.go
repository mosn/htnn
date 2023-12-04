package config

func GoSoPath() string {
	return "/etc/libgolang.so"
}

var rootNamespace = "istio-system"

func RootNamespace() string {
	return rootNamespace
}
