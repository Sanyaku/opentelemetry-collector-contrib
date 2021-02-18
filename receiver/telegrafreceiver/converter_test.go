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

package telegrafreceiver

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestConverter(t *testing.T) {
	tim := time.Now()

	tests := []struct {
		name          string
		metricsFn     func() (telegraf.Metric, error)
		separateField bool
		expectedErr   bool
		expectedFn    func() pdata.MetricSlice
	}{
		{
			name:          "gauge_int_with_one_field",
			separateField: false,
			metricsFn: func() (telegraf.Metric, error) {
				fields := map[string]interface{}{
					"available": uint64(39097651200),
				}

				return metric.New("mem", nil, fields, tim, telegraf.Gauge)
			},
			expectedFn: func() pdata.MetricSlice {
				metrics := pdata.NewMetricSlice()
				metrics.Append(newMetricIntGauge("mem_available", 39097651200, tim))
				return metrics
			},
		},
		{
			name:          "gauge_int_separate_field_with_one_field",
			separateField: true,
			metricsFn: func() (telegraf.Metric, error) {
				fields := map[string]interface{}{
					"available": uint64(39097651200),
				}

				return metric.New("mem", nil, fields, tim, telegraf.Gauge)
			},
			expectedFn: func() pdata.MetricSlice {
				metrics := pdata.NewMetricSlice()
				metrics.Append(newMetricIntGaugeWithSeparateField("mem", "available", 39097651200, tim))
				return metrics
			},
		},
		{
			name:          "gauge_double_with_one_field",
			separateField: false,
			metricsFn: func() (telegraf.Metric, error) {
				fields := map[string]interface{}{
					"available_percent": 54.505050,
				}

				return metric.New("mem", nil, fields, tim, telegraf.Gauge)
			},
			expectedFn: func() pdata.MetricSlice {
				metrics := pdata.NewMetricSlice()
				metrics.Append(newMetricDoubleGauge("mem_available_percent", 54.505050, tim))
				return metrics
			},
		},
		{
			name:          "gauge_double_separate_field_with_one_field",
			separateField: true,
			metricsFn: func() (telegraf.Metric, error) {
				fields := map[string]interface{}{
					"available_percent": 54.505050,
				}

				return metric.New("mem", nil, fields, tim, telegraf.Gauge)
			},
			expectedFn: func() pdata.MetricSlice {
				metrics := pdata.NewMetricSlice()
				metrics.Append(newMetricDoubleGaugeWithSeparateField("mem", "available_percent", 54.505050, tim))
				return metrics
			},
		},
		// TODO: work out unsorted fields coming from telegraf.FieldList()
		//
		// {
		// 	name:          "gauge_int_with_multiple_fields",
		// 	separateField: false,
		// 	metricsFn: func() (telegraf.Metric, error) {
		// 		fields := map[string]interface{}{
		// 			"available":    uint64(39097651200),
		// 			"free":         uint64(24322170880),
		// 			"total":        uint64(68719476736),
		// 			"used":         uint64(29621825536),
		// 			"used_percent": 43.10542941093445,
		// 		}

		// 		return metric.New("mem", nil, fields, tim, telegraf.Gauge)
		// 	},
		// 	expectedFn: func() pdata.MetricSlice {
		// 		metrics := pdata.NewMetricSlice()
		// 		metrics.Append(newMetricIntGauge("mem_available", 39097651200, tim))
		// 		metrics.Append(newMetricIntGauge("mem_free", 24322170880, tim))
		// 		metrics.Append(newMetricIntGauge("mem_total", 68719476736, tim))
		// 		metrics.Append(newMetricIntGauge("mem_used", 29621825536, tim))
		// 		metrics.Append(newMetricDoubleGauge("mem_used_percent", 43.10542941093445, tim))
		// 		return metrics
		// 	},
		// },
		// {
		// 	name:          "gauge_int_separate_field_with_multiple_fields",
		// 	separateField: true,
		// 	metricsFn: func() (telegraf.Metric, error) {
		// 		fields := map[string]interface{}{
		// 			"available": uint64(39097651200),
		// 			"free":      uint64(24322170880),
		// 		}

		// 		return metric.New("mem", nil, fields, tim, telegraf.Gauge)
		// 	},
		// 	expectedFn: func() pdata.MetricSlice {
		// 		metrics := pdata.NewMetricSlice()
		// 		metrics.Append(newMetricIntGaugeWithSeparateField("mem", "available", 39097651200, tim))
		// 		metrics.Append(newMetricIntGaugeWithSeparateField("mem", "free", 24322170880, tim))
		// 		return metrics
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := tt.metricsFn()
			require.NoError(t, err)

			mc := newConverter(tt.separateField)
			out, err := mc.Convert(m)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				expected := tt.expectedFn()
				actual := out.
					ResourceMetrics().At(0).
					InstrumentationLibraryMetrics().At(0).Metrics()
				assert.Equal(t, expected, actual)

				// if !assert.ElementsMatch(t, expected, actual) {
				// 	spew.Dump(tt.expectedFn())
				// 	println()
				// 	spew.Dump(out)
				// }
			}
		})
	}
}

func newMetricIntGauge(metric string, value int64, t time.Time) pdata.Metric {
	pm := pdata.NewMetric()
	pm.SetName(metric)
	pm.SetDataType(pdata.MetricDataTypeIntGauge)

	dps := pm.IntGauge().DataPoints()
	dps.Resize(1)
	dp := dps.At(0)
	dp.SetValue(value)
	dp.SetTimestamp(pdata.TimestampUnixNano(t.UnixNano()))
	return pm
}

func newMetricIntGaugeWithSeparateField(metric string, field string, value int64, t time.Time) pdata.Metric {
	pm := pdata.NewMetric()
	pm.SetName(metric)
	pm.SetDataType(pdata.MetricDataTypeIntGauge)

	dps := pm.IntGauge().DataPoints()
	dps.Resize(1)
	dp := dps.At(0)
	dp.SetValue(value)
	dp.SetTimestamp(pdata.TimestampUnixNano(t.UnixNano()))
	dp.LabelsMap().Insert(fieldLabel, field)
	return pm
}

func newMetricDoubleGauge(metric string, value float64, t time.Time) pdata.Metric {
	pm := pdata.NewMetric()
	pm.SetName(metric)
	pm.SetDataType(pdata.MetricDataTypeDoubleGauge)

	dps := pm.DoubleGauge().DataPoints()
	dps.Resize(1)
	dp := dps.At(0)
	dp.SetValue(value)
	dp.SetTimestamp(pdata.TimestampUnixNano(t.UnixNano()))
	return pm
}

func newMetricDoubleGaugeWithSeparateField(metric string, field string, value float64, t time.Time) pdata.Metric {
	pm := pdata.NewMetric()
	pm.SetName(metric)
	pm.SetDataType(pdata.MetricDataTypeDoubleGauge)

	dps := pm.DoubleGauge().DataPoints()
	dps.Resize(1)
	dp := dps.At(0)
	dp.SetValue(value)
	dp.SetTimestamp(pdata.TimestampUnixNano(t.UnixNano()))
	dp.LabelsMap().Insert(fieldLabel, field)
	return pm
}

// func sort(m pdata.MetricSlice) {
// }
