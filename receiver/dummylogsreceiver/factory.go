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
	"encoding/json"
	"fmt"
	"regexp"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

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

type Log struct {
	Date float64
	Log  string
}

type Server struct {
	port         int
	address      string
	protocol     string
	nextConsumer consumer.LogConsumer
	ctx          context.Context
}

func (s *Server) Start() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		io.Copy(buf, r.Body)
		var logs []Log
		text := buf.String()
		bytes := []byte(text)

		json.Unmarshal(bytes, &logs)
		tag := r.Header.Get("X-Sumo-Tag")

		regex := regexp.MustCompile(`\.containers\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\.log$`)
		details := regex.FindAllStringSubmatch(tag, -1)
		fmt.Printf("%q", details)

		flogs := pdata.NewLogs()
		resources := flogs.ResourceLogs()
		resources.Resize(1)
		resources.At(0).Logs().Resize(len(logs))

		for i, log := range logs {
			nlogs := resources.At(0).Logs().At(i)
			attributes := nlogs.Attributes()
			nlogs.SetBody(log.Log)
			// nlogs.SetTimestamp(log.Date)
			if len(details) == 1 && len(details[0]) == 4 {
				attributes.InsertString("pod_name", details[0][0])
				attributes.InsertString("namespace", details[0][1])
				attributes.InsertString("container_name", details[0][2])
				attributes.InsertString("docker_id", details[0][3])
			}
		}
		s.nextConsumer.ConsumeLogs(s.ctx, flogs)
	})

	log.Fatal(http.ListenAndServe(":24284", nil))
}

func (s *Server) handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Send a response back to person contacting us.
	conn.Write([]byte("Message received."))
	// Close the connection when you're done with it.
	conn.Close()
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
	server := Server{24284, "0.0.0.0", "tcp", consumer, ctx}
	go server.Start()
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
