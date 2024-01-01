module mosn.io/htnn/dev_your_plugin

go 1.20

require (
	github.com/envoyproxy/envoy v1.28.0
	mosn.io/htnn v0.1.0
)

require (
	github.com/cncf/xds/go v0.0.0-20231016030527-8bd2eac9fb4a // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace mosn.io/htnn => ../../
