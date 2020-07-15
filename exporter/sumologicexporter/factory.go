// Copyright 2019, OpenTelemetry Authors
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

// This file implements factory for Jaeger receiver.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

const (
	// The value of "type" key in configuration.
	typeStr = "sumologic"
)

// Factory is the factory for Jaeger legacy receiver.
type Factory struct {
	exporter component.LogExporter
}

type sumologicexporter struct {
	config          *Config
	logger          *zap.Logger
	metrics         []string
	metrics_counter int
	mutex           *sync.Mutex
}

// Type gets the type of the Receiver config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return configmodels.Type(typeStr)
}

// CreateDefaultConfig creates the default configuration for JaegerLegacy receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Exporter {
	return &Config{
		TypeVal: configmodels.Type(typeStr),
		NameVal: typeStr,
	}
}

// CustomUnmarshaler returns the custom function to handle the special settings
// used on the receiver.
func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return func(sourceViperSection *viper.Viper, intoCfg interface{}) error {
		return nil
	}
}

func (se *sumologicexporter) Shutdown(context.Context) error {
	return nil
}

func (se *sumologicexporter) Start(context.Context, component.Host) error {
	return nil
}

func (se *sumologicexporter) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	client := &http.Client{}

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		var buf bytes.Buffer
		body := bytes.Buffer{}
		for j := 0; j < ld.ResourceLogs().At(i).Logs().Len(); j++ {
			buf = bytes.Buffer{}
			body.WriteString(ld.ResourceLogs().At(i).Logs().At(j).Body())
			body.WriteString("\n")
			ld.ResourceLogs().At(i).Logs().At(j).Attributes().ForEach(func(k string, v pdata.AttributeValue) {
				buf.WriteString(k)
				buf.WriteString("=")
				buf.WriteString(v.StringVal())
				buf.WriteString(", ")
			})
		}
		req, _ := http.NewRequest("POST", se.config.Endpoint, bytes.NewBuffer(body.Bytes()))
		req.Header.Add("X-Sumo-Fields", buf.String())
		req.Header.Add("X-Sumo-Name", "otelcol")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		_, err := client.Do(req)

		if err != nil {
			fmt.Printf("Error during sending data to sumo: %q\n", err)
		}
	}
	return nil
}

func (se *sumologicexporter) AddPrometheusLine(name string, ts *timestamp.Timestamp, value float64, labels map[string]string) {
	labelsFmt := make([]string, len(labels))
	i := 0

	for name, label := range labels {
		labelsFmt[i] = fmt.Sprintf("%s=\"%s\"", name, label)
		i += 1
	}

	metric := fmt.Sprintf("%s{%s,_client=\"kubernetes\"} %f %d.%d", name, strings.Join(labelsFmt, ","), value, ts.GetSeconds(), ts.GetNanos())
	se.mutex.Lock()
	se.metrics[se.metrics_counter] = metric
	se.metrics_counter += 1
	if se.metrics_counter == len(se.metrics) {
		// Flush buffer
		client := &http.Client{}
		req, _ := http.NewRequest("POST", se.config.Endpoint, bytes.NewBuffer([]byte(metric)))
		req.Header.Add("X-Sumo-Name", "otelcol")
		req.Header.Add("Content-Type", "application/vnd.sumologic.prometheus")
		_, err := client.Do(req)

		if err != nil {
			fmt.Printf("Error during sending data to sumo: %q\n", err)
		}

		se.metrics_counter = 0
	}

	se.mutex.Unlock()
}

func (se *sumologicexporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	for i := 0; i < len(md.Metrics); i++ {
		metric := md.Metrics[i]
		metric_name := md.Resource.Labels["__name__"]

		for j := 0; j < len(metric.GetTimeseries()); j++ {
			serie := metric.GetTimeseries()[j]
			for k := 0; k < len(serie.GetPoints()); k++ {
				point := serie.GetPoints()[k]
				se.AddPrometheusLine(metric_name, point.GetTimestamp(), point.GetDoubleValue(), md.Resource.GetLabels())
			}
		}
	}
	return nil
}

func (se *sumologicexporter) ConsumeTraces(ctx context.Context, ld pdata.Traces) error {

	return nil
}

func (f *Factory) createExporter(
	config *Config,
) (component.LogExporter, error) {
	if f.exporter == nil {
		f.exporter = &sumologicexporter{
			config:  config,
			metrics: make([]string, 500),
			mutex:   &sync.Mutex{},
		}
	}

	return f.exporter, nil
}

func (f *Factory) CreateLogExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.LogExporter, error) {
	rCfg := cfg.(*Config)
	exporter, _ := f.createExporter(rCfg)

	return exporter.(component.LogExporter), nil
}

func (f *Factory) CreateMetricsExporter(
	logger *zap.Logger,
	cfg configmodels.Exporter,
) (component.MetricsExporterOld, error) {
	rCfg := cfg.(*Config)
	exporter, _ := f.createExporter(rCfg)

	return exporter.(component.MetricsExporterOld), nil
}

func (f *Factory) CreateTraceExporter(
	logger *zap.Logger,
	cfg configmodels.Exporter,
) (component.TraceExporterOld, error) {
	rCfg := cfg.(*Config)
	exporter, _ := f.createExporter(rCfg)

	return exporter.(component.TraceExporterOld), nil
}
