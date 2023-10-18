module mosn.io/moe

go 1.20

require (
	github.com/envoyproxy/envoy v1.27.1-0.20231017013410-9d787ffeeef3
	github.com/stretchr/testify v1.8.4
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/envoyproxy/envoy => ../envoy
