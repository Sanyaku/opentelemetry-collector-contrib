// Copyright 2019 OpenTelemetry Authors
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

package sourceprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
)

const (
	// The value of "type" key in configuration.
	typeStr = "source"

	defaultSource    = "traces"
	defaultCollector = ""

	defaultSourceName                = "%{namespace}.%{pod}.%{container}"
	defaultSourceCategory            = "%{namespace}/%{pod_name}"
	defaultSourceCategoryPrefix      = "kubernetes/"
	defaultSourceCategoryReplaceDash = "/"

	defaultAnnotationPrefix   = "pod_annotation_"
	defaultContainerKey       = "container"
	defaultNamespaceKey       = "namespace"
	defaultPodIDKey           = "pod_id"
	defaultPodKey             = "pod"
	defaultPodNameKey         = "pod_name"
	defaultPodTemplateHashKey = "pod_labels_pod-template-hash"
	defaultSourceHostKey      = "source_host"
)

// Factory is the factory for OpenCensus exporter.
type Factory struct {
}

var _ component.ProcessorFactory = (*Factory)(nil)

// Type gets the type of the Option config created by this factory.
func (Factory) Type() configmodels.Type {
	return typeStr
}

// CreateDefaultConfig creates the default configuration for processor.
func (Factory) CreateDefaultConfig() configmodels.Processor {
	return &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Source:                    defaultSource,
		Collector:                 defaultCollector,
		SourceName:                defaultSourceName,
		SourceCategory:            defaultSourceCategory,
		SourceCategoryPrefix:      defaultSourceCategoryPrefix,
		SourceCategoryReplaceDash: defaultSourceCategoryReplaceDash,

		AnnotationPrefix:   defaultAnnotationPrefix,
		ContainerKey:       defaultContainerKey,
		NamespaceKey:       defaultNamespaceKey,
		PodKey:             defaultPodKey,
		PodIDKey:           defaultPodIDKey,
		PodNameKey:         defaultPodNameKey,
		PodTemplateHashKey: defaultPodTemplateHashKey,
		SourceHostKey:      defaultSourceHostKey,
	}
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateTraceProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	cfg configmodels.Processor) (component.TraceProcessor, error) {

	oCfg := cfg.(*Config)
	return newSourceTraceProcessor(nextConsumer, oCfg)
}

// CreateMetricsProcessor creates a metric processor based on this config.
func (f *Factory) CreateMetricsProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	_ consumer.MetricsConsumer,
	_ configmodels.Processor) (component.MetricsProcessor, error) {
	// Span Processor does not support Metrics.
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateTraceProcessor creates a trace processor based on this config.
func (f *Factory) CreateLogProcessor(
	_ context.Context,
	_ component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.LogConsumer) (component.LogProcessor, error) {

	oCfg := cfg.(*Config)
	return newSourceLogsProcessor(nextConsumer, oCfg)
}
