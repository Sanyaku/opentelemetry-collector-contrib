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
	logKey string = "log"
)

type sumologicexporter struct {
	config          *Config
	metadataRegexes []*regexp.Regexp
	client          *http.Client
	f               filterator
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
		f: newFilterator(cfg.MetadataFields),
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
func (se *sumologicexporter) filterMetadata(attributes pdata.AttributeMap, filterOut bool) map[string]string {
	returnValue := make(map[string]string)
	attributes.ForEach(func(k string, v pdata.AttributeValue) {
		skip := !filterOut
		for j := 0; j < len(se.metadataRegexes); j++ {
			if se.metadataRegexes[j].MatchString(k) {
				skip = false
				if filterOut {
					return
				}
			}
		}

		if skip {
			return
		}

		returnValue[k] = v.StringVal()
	})

	return returnValue
}

type Filterator struct {
	metadataRegexes []*regexp.Regexp
}

func (f Filterator) Filter(attributes pdata.AttributeMap) map[string]string {
	return nil
}

func (f Filterator) FilterOut(attributes pdata.AttributeMap) map[string]string {
	// TODO filter out
	return nil
}

func (se *sumologicexporter) GetMetadata(attributes pdata.AttributeMap) string {
	attr := se.filterator.Filter(attributes)
	// attrs := se.filterMetadata(attributes, false)

	metadata := make([]string, 0, len(attrs))

	for k := range attrs {
		metadata = append(metadata, fmt.Sprintf("%s=%s", k, attrs[k]))
	}
	sort.Strings(metadata)

	return strings.Join(metadata, ", ")
}

// pushLogsData groups data with common metadata uses sendAndPushErrors to send data to sumologic
func (se *sumologicexporter) pushLogsData(ctx context.Context, ld pdata.Logs) (droppedTimeSeries int, err error) {
	var (
		errs             []error
		previousMetadata string
		currentMetadata  string
	)

	const maxBufferSize = 100
	s := newSender(se.client, se.config, se.filterator, maxBufferSize)

	// Iterate over ResourceLogs
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)

		// iterate over InstrumentationLibraryLogs
		for j := 0; j < resource.InstrumentationLibraryLogs().Len(); j++ {
			library := resource.InstrumentationLibraryLogs().At(j)

			// iterate over Logs
			for k := 0; k < library.Logs().Len(); k++ {
				log := library.Logs().At(k)
				currentMetadata = se.GetMetadata(log.Attributes())

				// If metadate differs from currently buffered, flush the buffer
				if currentMetadata != previousMetadata && previousMetadata != "" {
					// se.sendAndPushErrors(&buffer, previousMetadata, &droppedTimeSeries, &errs)
					n, err := s.Send(previousMetadata)
					if err != nil {
						droppedTimeSeries += n
						errs = append(errs, err)
					}
				}

				// assign metadata
				previousMetadata = currentMetadata

				s.AppendToBuffer(log)

				// Flush buffer to avoid overlow
				if len(buffer) == maxBufferSize {
					s.Send(&buffer, previousMetadata, &droppedTimeSeries, &errs)
					n, err := s.Send(previousMetadata)
					if err != nil {
						droppedTimeSeries += n
						errs = append(errs, err)
					}
				}
			}
		}
	}

	// Flush pending logs
	se.sendAndPushErrors(&buffer, previousMetadata, &droppedTimeSeries, &errs)

	return droppedTimeSeries, componenterror.CombineErrors(errs)
}
