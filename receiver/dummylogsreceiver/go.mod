module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dummylogsreceiver

go 1.14

require (
	github.com/gogo/protobuf v1.3.1
	github.com/grpc-ecosystem/grpc-gateway v1.14.5
	github.com/spf13/viper v1.7.0
	go.opentelemetry.io/collector v0.4.0
	go.uber.org/zap v1.15.0
	google.golang.org/genproto v0.0.0-20200408120641-fbb3ad325eb7
	google.golang.org/grpc v1.29.1
)

replace go.opentelemetry.io/collector => github.com/SumoLogic/opentelemetry-collector v0.2.7-0.20200617091426-534450443e58
