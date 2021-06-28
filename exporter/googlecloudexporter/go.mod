module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter

go 1.14

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.5
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.19.0
	github.com/census-instrumentation/opencensus-proto v0.3.0
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/onsi/ginkgo v1.14.1 // indirect
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/stretchr/testify v1.7.0
	go.opencensus.io v0.23.0
	go.opentelemetry.io/collector v0.29.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/trace v0.19.0
	go.uber.org/zap v1.17.0
	google.golang.org/api v0.48.0
	google.golang.org/genproto v0.0.0-20210604141403-392c879c8b08
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/square/go-jose.v2 v2.5.1 // indirect
)
