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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	logKey        string = "log"
	maxBufferSize int    = 100
)

type sumologicexporter struct {
	config          *Config
	metadataRegexes []*regexp.Regexp
	client          *http.Client
}

type sender struct {
	buffer            []pdata.LogRecord
	previousMetadata  string
	currentMetadata   string
	errs              []error
	droppedTimeSeries int
}

func newLogsExporter(
	cfg *Config,
) (component.LogsExporter, error) {
	se, err := initExporter(cfg)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewLogsExporter(
		cfg,
		se.pushLogsData,
		// Disable exporterhelper Timeout, since we are using a custom mechanism
		// within exporter itself
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
	)
}

func initExporter(cfg *Config) (*sumologicexporter, error) {
	se := &sumologicexporter{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.TimeoutSettings.Timeout,
		},
	}

	err := se.refreshMetadataRegexes()
	if err != nil {
		return nil, err
	}

	return se, nil
}

func (se *sumologicexporter) refreshMetadataRegexes() error {
	cfg := se.config
	metadataRegexes := make([]*regexp.Regexp, len(cfg.MetadataFields))
	for i := 0; i < len(cfg.MetadataFields); i++ {

		regex, err := regexp.Compile(cfg.MetadataFields[i])
		if err != nil {
			return err
		}

		metadataRegexes[i] = regex
	}

	se.metadataRegexes = metadataRegexes
	return nil
}

// filterMetadata returns map of attributes which are (or are not, it depends on filterOut argument) metadata
// for filterOut equals false -> return all keys which match at least one regular exporession
// for filterOut equals true -> return all keys which do not match any regular expression
func (se *sumologicexporter) filterMetadata(attributes pdata.AttributeMap, filterOut bool) map[string]string {
	returnValue := make(map[string]string)
	attributes.ForEach(func(k string, v pdata.AttributeValue) {
		switch filterOut {
		case true:
			for _, regex := range se.metadataRegexes {
				if regex.MatchString(k) {
					return
				}
			}
			returnValue[k] = v.StringVal()
		case false:
			for _, regex := range se.metadataRegexes {
				if regex.MatchString(k) {
					returnValue[k] = v.StringVal()
					return
				}
			}
		}
	})

	return returnValue
}

// GetMetadata builds string which represents metadata in alphabetical order
func (se *sumologicexporter) GetMetadata(attributes pdata.AttributeMap) string {
	attrs := se.filterMetadata(attributes, false)
	metadata := make([]string, 0, len(attrs))

	for k := range attrs {
		metadata = append(metadata, fmt.Sprintf("%s=%s", k, attrs[k]))
	}
	sort.Strings(metadata)

	return strings.Join(metadata, ", ")
}

// This function tries to send data and eventually appends error in case of failure
// It modifies buffer, droppedTimeSeries and errs
func (se *sumologicexporter) sendAndPushErrors(sdr *sender) {

	err := se.sendLogs(sdr.buffer, sdr.previousMetadata)
	if err != nil {
		sdr.droppedTimeSeries += len(sdr.buffer)
		sdr.errs = append(sdr.errs, err)
	}

	sdr.buffer = (sdr.buffer)[:0]
}

// pushLogsData groups data with common metadata uses sendAndPushErrors to send data to sumologic
func (se *sumologicexporter) pushLogsData(ctx context.Context, ld pdata.Logs) (droppedTimeSeries int, err error) {
	sdr := &sender{
		buffer: make([]pdata.LogRecord, 0, maxBufferSize),
	}

	// Iterate over ResourceLogs
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)

		// iterate over InstrumentationLibraryLogs
		for j := 0; j < resource.InstrumentationLibraryLogs().Len(); j++ {
			library := resource.InstrumentationLibraryLogs().At(j)

			// iterate over Logs
			for k := 0; k < library.Logs().Len(); k++ {
				log := library.Logs().At(k)
				sdr.currentMetadata = se.GetMetadata(log.Attributes())

				// If metadate differs from currently buffered, flush the buffer
				if sdr.currentMetadata != sdr.previousMetadata && sdr.previousMetadata != "" {
					se.sendAndPushErrors(sdr)
				}

				// assign metadata
				sdr.previousMetadata = sdr.currentMetadata

				// add log to the buffer
				sdr.buffer = append(sdr.buffer, log)

				// Flush buffer to avoid overlow
				if len(sdr.buffer) == maxBufferSize {
					se.sendAndPushErrors(sdr)
				}
			}
		}
	}

	// Flush pending logs
	se.sendAndPushErrors(sdr)

	return sdr.droppedTimeSeries, componenterror.CombineErrors(sdr.errs)
}

// appendAndSend appends line to the body and eventually sends data to avoid exceeding the request limit
func (se *sumologicexporter) appendAndSend(line string, pipeline string, body *strings.Builder, fields string) error {
	var err error

	if body.Len() > 0 && body.Len()+len(line) > se.config.MaxRequestBodySize {
		err = se.send(LogsPipeline, body.String(), fields)
		body.Reset()
	}

	if body.Len() > 0 {
		// Do not add newline if the body is empty
		body.WriteString("\n")
	}

	body.WriteString(line)
	return err
}

func (se *sumologicexporter) sendLogsTextFormat(buffer []pdata.LogRecord, fields string) error {
	body := strings.Builder{}
	var errs []error

	for j := 0; j < len(buffer); j++ {
		err := se.appendAndSend(buffer[j].Body().StringVal(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := se.send(LogsPipeline, body.String(), fields)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

func (se *sumologicexporter) sendLogsJSONFormat(buffer []pdata.LogRecord, fields string) error {
	body := strings.Builder{}
	var errs []error

	for j := 0; j < len(buffer); j++ {
		data := se.filterMetadata(buffer[j].Attributes(), true)
		data[logKey] = buffer[j].Body().StringVal()

		nextLine, err := json.Marshal(data)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = se.appendAndSend(bytes.NewBuffer(nextLine).String(), LogsPipeline, &body, fields)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := se.send(LogsPipeline, body.String(), fields)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}
	return nil
}

func (se *sumologicexporter) sendLogs(buffer []pdata.LogRecord, fields string) error {
	switch se.config.LogFormat {
	case TextFormat:
		return se.sendLogsTextFormat(buffer, fields)
	case JSONFormat:
		return se.sendLogsJSONFormat(buffer, fields)
	default:
		return errors.New("Unexpected log format")
	}
}

// Send sends data to sumologic
func (se *sumologicexporter) send(pipeline string, body string, fields string) error {
	// Add headers
	req, err := http.NewRequest("POST", se.config.URL, strings.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Add("X-Sumo-Client", se.config.Client)

	if len(se.config.SourceHost) > 0 {
		req.Header.Add("X-Sumo-Host", se.config.SourceHost)
	}

	if len(se.config.SourceName) > 0 {
		req.Header.Add("X-Sumo-Name", se.config.SourceName)
	}

	if len(se.config.SourceCategory) > 0 {
		req.Header.Add("X-Sumo-Category", se.config.SourceCategory)
	}

	switch pipeline {
	case LogsPipeline:
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("X-Sumo-Fields", fields)
	case MetricsPipeline:
		// ToDo: Implement metrics pipeline
		return errors.New("Current sender version doesn't support metrics")
	default:
		return errors.New("Unexpected pipeline")
	}

	_, err = se.client.Do(req)
	// ToDo: Add retries mechanism
	if err != nil {
		return err
	}
	return nil
}
