// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	// The value of "type" key in configuration.
	typeStr = "sumologic"
)

// NewFactory returns a new factory for the sumologic processor.
func NewFactory() component.ExporterFactory {
	return exporterhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		exporterhelper.WithLogs(createLogsExporter),
	)
}

func createDefaultConfig() configmodels.Exporter {
	qs := exporterhelper.CreateDefaultQueueSettings()
	qs.Enabled = false

	return &Config{
		ExporterSettings: configmodels.ExporterSettings{
			TypeVal: configmodels.Type(typeStr),
			NameVal: typeStr,
		},

		Compress:           DefaultCompress,
		CompressEncoding:   DefaultCompressEncoding,
		MaxRequestBodySize: DefaultMaxRequestBodySize,
		LogFormat:          DefaultLogFormat,
		MetricFormat:       DefaultMetricFormat,
		SourceCategory:     DefaultSourceCategory,
		SourceName:         DefaultSourceName,
		SourceHost:         DefaultSourceHost,
		Client:             DefaultClient,

		TimeoutSettings: CreateDefaultTimeoutSettings(),
		RetrySettings:   exporterhelper.CreateDefaultRetrySettings(),
		QueueSettings:   qs,
	}
}

func createLogsExporter(
	_ context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.LogsExporter, error) {
	exp, err := newLogsExporter(cfg.(*Config))
	if err != nil {
		return nil, err
	}

	return exp, nil
}
