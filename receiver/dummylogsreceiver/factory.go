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
	"io"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dummylogsreceiver/prompb"
)

const (
	// The value of "type" key in configuration.
	typeStr = "dummylogs"
)

// Factory is the factory for Jaeger legacy receiver.
type Factory struct {
	receiver *dummylogsReceiver
}

type Log struct {
	Date float64
	Log  string
}

type Server struct {
	port     int
	address  string
	protocol string
	receiver *dummylogsReceiver
	ctx      context.Context
}

func (s *Server) StartLogsServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		buf := new(strings.Builder)
		io.Copy(buf, r.Body)
		var logs []map[string]interface{}
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

		flogs.ResourceLogs().At(0).Resource().InitEmpty()
		attributes := flogs.ResourceLogs().At(0).Resource().Attributes()
		if len(details) == 1 && len(details[0]) == 5 {
			attributes.InsertString("k8s.pod.name", details[0][1])
			attributes.InsertString("pod_name", details[0][1])
			attributes.InsertString("namespace", details[0][2])
			attributes.InsertString("container_name", details[0][3])
			attributes.InsertString("docker_id", details[0][4])
		}

		for i, log := range logs {
			nlogs := resources.At(0).Logs().At(i)
			for name, value := range log {
				nlogs.Attributes().InsertString(name, value.(string))
			}
		}
		s.receiver.LogConsumer.ConsumeLogs(s.ctx, flogs)
	})

	log.Fatal(http.ListenAndServe(":24284", mux))
}

func (s *Server) StartMetricsServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contentLength, _ := strconv.Atoi(r.Header.Get("Content-Length"))
		in := make([]byte, contentLength)
		count, _ := r.Body.Read(in)
		if count != contentLength {
			log.Printf("Received more data than processed")
			return
		}
		in, err := snappy.Decode(nil, in)
		if err != nil {
			log.Printf("Failed to decode request %s", err)
			return
		}
		request := &prompb.WriteRequest{}
		if err := proto.Unmarshal(in, request); err != nil {
			log.Printf("Failed to parse prometheus request %s", err)
		} else {
			for _, metric := range request.Timeseries {
				md := consumerdata.MetricsData{}

				// Inject metric labels
				md.Resource = &resourcepb.Resource{
					Labels: map[string]string{},
				}
				for _, label := range metric.Labels {
					md.Resource.Labels[label.Name] = label.Value
				}

				md.Node = &commonpb.Node{
					Identifier: &commonpb.ProcessIdentifier{HostName: strings.Split(md.Resource.Labels["instance"], ":")[0]},
				}

				md.Metrics = []*metricspb.Metric{
					{
						MetricDescriptor: &metricspb.MetricDescriptor{
							Name: md.Resource.Labels["__name__"],
							Type: metricspb.MetricDescriptor_GAUGE_DOUBLE,
						},
						Timeseries: make([]*metricspb.TimeSeries, 1),
					},
				}

				md.Metrics[0].Timeseries[0] = &metricspb.TimeSeries{
					Points: make([]*metricspb.Point, len(metric.Samples)),
				}

				for j, point := range metric.Samples {
					md.Metrics[0].Timeseries[0].Points[j] = &metricspb.Point{
						Value:     &metricspb.Point_DoubleValue{DoubleValue: point.Value},
						Timestamp: &timestamppb.Timestamp{Seconds: point.Timestamp / 1e3, Nanos: int32((point.Timestamp % 1e3) * 1e6)},
					}
				}
				s.receiver.MetricsConsumer.ConsumeMetricsData(s.ctx, md)
			}
		}
	})

	log.Fatal(http.ListenAndServe(":24285", mux))
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
	config          *Config
	logger          *zap.Logger
	LogConsumer     consumer.LogConsumer
	MetricsConsumer consumer.MetricsConsumerOld
}

// Type gets the type of the Receiver config created by this factory.
func (f *Factory) Type() configmodels.Type {
	return configmodels.Type(typeStr)
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

	if f.receiver != nil {
		return f.receiver, nil
	}

	r := &dummylogsReceiver{
		config: config,
	}

	f.receiver = r

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

	server := Server{24284, "0.0.0.0", "tcp", receiver.(*dummylogsReceiver), ctx}
	go server.StartLogsServer()
	return receiver, nil
}

func (f *Factory) CreateMetricsReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumerOld,
) (component.MetricsReceiver, error) {
	rCfg := cfg.(*Config)
	receiver, _ := f.createReceiver(rCfg)
	receiver.(*dummylogsReceiver).MetricsConsumer = nextConsumer

	server := Server{24285, "0.0.0.0", "tcp", receiver.(*dummylogsReceiver), ctx}
	go server.StartMetricsServer()
	return receiver, nil
}

func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	logger *zap.Logger,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumerOld,
) (component.TraceReceiver, error) {
	// No trace receiver for now.
	return nil, configerror.ErrDataTypeIsNotSupported
}
