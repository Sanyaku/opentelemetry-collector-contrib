module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter

go 1.14

require (
	github.com/spf13/viper v1.7.0
	go.opentelemetry.io/collector v0.4.0
	go.uber.org/zap v1.15.0
)

replace go.opentelemetry.io/collector => github.com/SumoLogic/opentelemetry-collector v0.2.7-0.20200617091426-534450443e58
