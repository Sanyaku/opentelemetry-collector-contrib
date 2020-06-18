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
	"net/http"

	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

const (
	// The value of "type" key in configuration.
	typeStr = "sumologic"
)

// Factory is the factory for Jaeger legacy receiver.
type Factory struct {
}

type sumologicexporter struct {
	config *Config
	logger *zap.Logger
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
		client.Do(req)
	}
	return nil
}

func (f *Factory) createExporter(
	config *Config,
) (component.LogExporter, error) {
	r := &sumologicexporter{
		config: config,
	}

	return r, nil
}

func (f *Factory) CreateLogExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	cfg configmodels.Exporter,
) (component.LogExporter, error) {
	rCfg := cfg.(*Config)
	exporter, _ := f.createExporter(rCfg)

	return exporter, nil
}
