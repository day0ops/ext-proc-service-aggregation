module github.com/day0ops/ext-proc-service-aggregation

go 1.22

require (
	github.com/envoyproxy/go-control-plane v0.12.0
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.6.0
	google.golang.org/grpc v1.64.0
)

require (
	github.com/cncf/xds/go v0.0.0-20240318125728-8a4994d93e50 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/envoyproxy/go-control-plane => github.com/solo-io/go-control-plane-fork-v2 v0.0.0-20231207195634-98d37ef9a43e
