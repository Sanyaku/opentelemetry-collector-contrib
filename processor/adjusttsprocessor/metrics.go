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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/collector/obsreport"
)

var (
	mSpansCorrected        = stats.Int64("processor_adjustts_spans_corrected", "Spans that had timestamp corrected", stats.UnitDimensionless)
	mSpansNotCorrected     = stats.Int64("processor_adjustts_spans_not_corrected", "Spans that did not have timestamp corrected", stats.UnitDimensionless)
	mSpansMissingReceiveTs = stats.Int64("processor_adjustts_spans_missing_receive_ts", "Spans that have missed receive timestamp", stats.UnitDimensionless)
	mSpansMissingExportTs  = stats.Int64("processor_adjustts_spans_missing_export_ts", "Spans that have missed export timestamp", stats.UnitDimensionless)
	mSpansInvalidExportTs  = stats.Int64("processor_adjustts_spans_invalid_export_ts", "Spans that have invalid export timestamp", stats.UnitDimensionless)
	mCorrectionHistogram   = stats.Int64("processor_adjustts_correction", "Correction factor", stats.UnitSeconds)
)

// MetricViews return the metrics views according to given telemetry level.
func MetricViews() []*view.View {
	legacyViews := []*view.View{
		{
			Name:        mSpansCorrected.Name(),
			Measure:     mSpansCorrected,
			Description: mSpansCorrected.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        mSpansNotCorrected.Name(),
			Measure:     mSpansNotCorrected,
			Description: mSpansNotCorrected.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        mSpansMissingReceiveTs.Name(),
			Measure:     mSpansMissingReceiveTs,
			Description: mSpansMissingReceiveTs.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        mSpansMissingExportTs.Name(),
			Measure:     mSpansMissingExportTs,
			Description: mSpansMissingExportTs.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        mSpansInvalidExportTs.Name(),
			Measure:     mSpansInvalidExportTs,
			Description: mSpansInvalidExportTs.Description(),
			Aggregation: view.LastValue(),
		},
		{
			Name:        mCorrectionHistogram.Name(),
			Measure:     mCorrectionHistogram,
			Description: mCorrectionHistogram.Description(),
			Aggregation: view.Distribution(-2592000, -86400, -21600, -3600, -1800, -600, -60, -10, -5, 0, 5, 10, 60, 600, 1800, 3600, 21600, 86400, 2592000),
		},
	}

	return obsreport.ProcessorMetricViews(string(typeStr), legacyViews)
}
