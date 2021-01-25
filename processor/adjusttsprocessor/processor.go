// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adjusttsprocessor

import (
	"context"
	"strconv"
	"time"

	"go.opencensus.io/stats"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

// adjustTsProcessor fixes grossly incorrect timestamp reported in spans
type adjustTsProcessor struct {
	config Config
	logger *zap.Logger

	// threshold specifies minimum duration difference above which the correction happens
	threshold time.Duration
}

var (
	// Anything outside these dates is considered not a misconfigured clock but rather a problem with exported timestamp
	minSaneExportTs = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	maxSaneExportTs = time.Now().Add(10 * time.Hour * 24 * 365)
)

// newAdjustTsProcessor returns a new processor.
func newAdjustTsProcessor(logger *zap.Logger, config Config) *adjustTsProcessor {
	return &adjustTsProcessor{
		logger:    logger,
		config:    config,
		threshold: config.Threshold,
	}
}

func (atsp *adjustTsProcessor) ProcessTraces(ctx context.Context, td pdata.Traces) (pdata.Traces, error) {
	cc, ok := client.FromContext(ctx)
	if !ok {
		stats.Record(context.Background(), mSpansMissingExportTs.M(int64(td.SpanCount())))
	} else {
		atsp.adjustExportTimestamp(td, cc.ReceiveTS)
	}

	return td, nil
}

func adjustTimestamp(tun pdata.TimestampUnixNano, delta time.Duration) pdata.TimestampUnixNano {
	return tun + pdata.TimestampUnixNano(delta.Nanoseconds())
}

func (atsp *adjustTsProcessor) adjustSpan(span pdata.Span, receiveTs time.Time, exportTs *time.Time) {
	if exportTs == nil {
		return
	}

	delta := receiveTs.Sub(*exportTs)
	if delta > atsp.threshold || delta < -atsp.threshold {
		span.SetStartTime(adjustTimestamp(span.StartTime(), delta))
		span.SetEndTime(adjustTimestamp(span.EndTime(), delta))
		stats.Record(context.Background(), mSpansCorrected.M(1))
		stats.Record(context.Background(), mCorrectionHistogram.M(int64(delta.Seconds())))
	} else {
		stats.Record(context.Background(), mSpansNotCorrected.M(1))
	}
}

func sanitizeTimestamp(ts *time.Time) *time.Time {
	if ts.After(minSaneExportTs) && ts.Before(maxSaneExportTs) {
		return ts
	}
	return nil
}

func extractExportTs(val pdata.AttributeValue) *time.Time {
	var timeMilis int64
	if val.Type() == pdata.AttributeValueINT {
		timeMilis = val.IntVal()
	} else if val.Type() == pdata.AttributeValueDOUBLE {
		timeMilis = int64(val.DoubleVal())
	} else if val.Type() == pdata.AttributeValueSTRING {
		asInt, err := strconv.Atoi(val.StringVal())
		if err != nil && asInt > 0 {
			timeMilis = int64(asInt)
		} else {
			return nil
		}
	}

	timeSeconds := timeMilis / 1_000
	timeNanos := (timeMilis - timeSeconds*1_000) * 1_000_000
	ts := time.Unix(timeSeconds, timeNanos)
	return sanitizeTimestamp(&ts)
}

func (atsp *adjustTsProcessor) adjustExportTimestamp(traces pdata.Traces, receiveTs time.Time) pdata.Traces {
	rss := traces.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		res := rss.At(i).Resource()
		resExportTs, resExportTsFound := res.Attributes().Get(AttributeSumoTelemetryExportTS)
		var exportTs *time.Time
		if resExportTsFound {
			exportTs = extractExportTs(resExportTs)
		}
		for j := 0; j < rss.At(i).InstrumentationLibrarySpans().Len(); j++ {
			spans := rss.At(i).InstrumentationLibrarySpans().At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				spanAttrs := spans.At(k).Attributes()
				spanExportTs, spanExportTsFound := spanAttrs.Get(AttributeSumoTelemetryExportTS)
				if spanExportTsFound {
					exportTs = extractExportTs(spanExportTs)
				}

				if exportTs == nil {
					if spanExportTsFound || resExportTsFound {
						stats.Record(context.Background(), mSpansInvalidExportTs.M(1))
					} else {
						stats.Record(context.Background(), mSpansMissingExportTs.M(1))
					}
				} else {
					spanAttrs.Delete(AttributeSumoTelemetryExportTS)
				}

				atsp.adjustSpan(spans.At(k), receiveTs, exportTs)
			}
		}
		if resExportTsFound {
			res.Attributes().Delete(AttributeSumoTelemetryExportTS)
		}
	}

	return traces
}

func (atsp *adjustTsProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}
