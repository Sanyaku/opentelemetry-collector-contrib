// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicexporter

import (
	"time"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for Sumo Logic exporter.
type Config struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// Global options
	// Unique URL generated for your HTTP Source.
	// This is the address to send data to.
	// url: "https://collectors.sumologic.com/receiver/v1/http/<UniqueHTTPCollectorCode>"
	URL string `mapstructure:"url"`
	// Option to enable compression (default true)
	Compress bool `mapstructure:"compress"`
	// Compression encoding format, either gzip or deflate (default gzip)
	CompressEncoding CompressEncodingType `mapstructure:"compress_encoding"`
	// Max HTTP request body size in bytes before compression (if applied).
	// By default 1MB is recommended.
	MaxRequestBodySize int `mapstructure:"max_request_body_size"`

	// Logs related configuration
	// Format to post logs into Sumo. (default json)
	//   * text - Logs will appear in Sumo Logic in text format.
	//   * json - Logs will appear in Sumo Logic in json format.
	LogFormat LogFormatType `mapstructure:"log_format"`

	// Metrics related configuration
	// The format of metrics you will be sending, either graphite or carbon2 or prometheus (Default is carbon2)
	MetricFormat MetricFormatType `mapstructure:"metric_format"`

	// List of regexes for attributes which should be send as metadata
	MetadataAttributes []string `mapstructure:"metadata_attributes"`

	// Sumo specific options
	// Desired source category.
	// Useful if you want to override the source category configured for the source.
	SourceCategory string `mapstructure:"source_category"`
	// Desired source name.
	// Useful if you want to override the source name configured for the source.
	SourceName string `mapstructure:"source_name"`
	// Desired host name.
	// Useful if you want to override the source host configured for the source.
	SourceHost string `mapstructure:"source_host"`
	// Name of the client
	Client string `mapstructure:"client"`

	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	exporterhelper.RetrySettings   `mapstructure:"retry_on_failure"`
}

// CreateDefaultTimeoutSettings returns default timeout settings
func CreateDefaultTimeoutSettings() exporterhelper.TimeoutSettings {
	return exporterhelper.TimeoutSettings{
		Timeout: defaultTimeout,
	}
}

// LogFormatType represents log_format
type LogFormatType string

// MetricFormatType represents metric_format
type MetricFormatType string

// PipelineType represents type of the pipeline
type PipelineType string

// CompressEncodingType represents type of the pipeline
type CompressEncodingType string

const (
	// TextFormat represents log_format: text
	TextFormat LogFormatType = "text"
	// JSONFormat represents log_format: json
	JSONFormat LogFormatType = "json"
	// GraphiteFormat represents metric_format: text
	GraphiteFormat MetricFormatType = "graphite"
	// Carbon2Format represents metric_format: json
	Carbon2Format MetricFormatType = "carbon2"
	// PrometheusFormat represents metric_format: json
	PrometheusFormat MetricFormatType = "prometheus"
	// GZIPCompression represents compress_encoding: gzip
	GZIPCompression CompressEncodingType = "gzip"
	// DeflateCompression represents compress_encoding: deflate
	DeflateCompression CompressEncodingType = "deflate"
	// NoCompression represents no compression (text)
	NoCompression CompressEncodingType = "text"
	// MetricsPipeline represents metrics pipeline
	MetricsPipeline PipelineType = "metrics"
	// LogsPipeline represents metrics pipeline
	LogsPipeline PipelineType = "logs"
	// defaultTimeout
	defaultTimeout time.Duration = 55 * time.Second
	// DefaultCompress defines default Compress
	DefaultCompress bool = true
	// DefaultCompressEncoding defines default CompressEncoding
	DefaultCompressEncoding CompressEncodingType = "gzip"
	// DefaultMaxRequestBodySize defines default MaxRequestBodySize in bytes
	DefaultMaxRequestBodySize int = 20 * 1024 * 1024
	// DefaultLogFormat defines default LogFormat
	DefaultLogFormat LogFormatType = JSONFormat
	// DefaultMetricFormat defines default MetricFormat
	DefaultMetricFormat MetricFormatType = Carbon2Format
	// DefaultSourceCategory defines default SourceCategory
	DefaultSourceCategory string = ""
	// DefaultSourceName defines default SourceName
	DefaultSourceName string = ""
	// DefaultSourceHost defines default SourceHost
	DefaultSourceHost string = ""
	// DefaultClient defines default Client
	DefaultClient string = "otelcol"
)
