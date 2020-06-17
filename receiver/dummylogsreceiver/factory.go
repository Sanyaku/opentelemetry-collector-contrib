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

package dummylogsreceiver

// This file implements factory for Jaeger receiver.

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

const (
	// The value of "type" key in configuration.
	typeStr = "dummylogs"
)

// Factory is the factory for Jaeger legacy receiver.
type Factory struct {
}

type dummylogsReceiver struct {
	config      *Config
	logger      *zap.Logger
	LogConsumer consumer.LogConsumer
}

// Type gets the type of the Receiver config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return configmodels.Type(typeStr)
}

func (f *Factory) generator(ctx context.Context, consumer consumer.LogConsumer) {
	for {
		logs := pdata.NewLogs()
		resources := logs.ResourceLogs()
		resources.Resize(1)
		resources.At(0).Logs().Resize(1)
		nlogs := resources.At(0).Logs().At(0)
		nlogs.SetBody("My example log")
		nlogs.Attributes().InsertString("ip", "100.96.1.4")
		consumer.ConsumeLogs(ctx, logs)
		time.Sleep(1e9)
	}
}

// CreateDefaultConfig creates the default configuration for JaegerLegacy receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
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

func (kr *dummylogsReceiver) Shutdown(context.Context) error {
	return nil
}

func (kr *dummylogsReceiver) Start(context.Context, component.Host) error {
	return nil
}

func (f *Factory) createReceiver(
	config *Config,
) (component.LogReceiver, error) {

	r := &dummylogsReceiver{
		config: config,
	}

	return r, nil
}

func (f *Factory) CreateLogReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.LogConsumer,
) (component.LogReceiver, error) {
	rCfg := cfg.(*Config)
	receiver, _ := f.createReceiver(rCfg)
	receiver.(*dummylogsReceiver).LogConsumer = nextConsumer

	go f.generator(ctx, nextConsumer)
	return receiver, nil
}
