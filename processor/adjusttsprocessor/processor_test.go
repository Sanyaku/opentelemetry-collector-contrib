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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewDevelopment()
	// Since export timestamp is having millisecond precision, lets keep it simple
	baseTs    = time.Now().Round(time.Millisecond)
)

// Common structure for the test cases.
type testCase struct {
	name           string
	receiveTs      time.Time
	inputTraces    pdata.Traces
	expectedTraces pdata.Traces
}

func runIndividualTestCase(t *testing.T, tt testCase, tp component.TracesProcessor) {
	t.Run(tt.name, func(t *testing.T) {
		ctx := client.NewContext(context.Background(), &client.Client{
			ReceiveTS: tt.receiveTs,
		})
		assert.NoError(t, tp.ConsumeTraces(ctx, tt.inputTraces))
		prepareTd(tt.inputTraces)
		prepareTd(tt.expectedTraces)
		assert.EqualValues(t, tt.expectedTraces, tt.inputTraces)
	})
}

func TestCases(t *testing.T) {
	badExportTs := time.Date(1984, 0, 0, 0, 0, 0, 0, time.UTC)

	testCases := []testCase{
		{
			name:           "Within threshold",
			receiveTs:      baseTs,
			inputTraces:    withSpanExportTs(baseTs, simpleTraces(2*time.Second)),
			expectedTraces: simpleTraces(2 * time.Second),
		},
		{
			name:           "Below threshold",
			receiveTs:      baseTs,
			inputTraces:    withSpanExportTs(baseTs.Add(-20*time.Second), simpleTraces(-18*time.Second)),
			expectedTraces: simpleTraces(2 * time.Second),
		},
		{
			name:           "Above threshold",
			receiveTs:      baseTs,
			inputTraces:    withSpanExportTs(baseTs.Add(20*time.Second), simpleTraces(22*time.Second)),
			expectedTraces: simpleTraces(2 * time.Second),
		},
		{
			name:           "Within threshold, old traces",
			receiveTs:      baseTs,
			inputTraces:    withSpanExportTs(baseTs, simpleTraces(-20*time.Second)),
			expectedTraces: simpleTraces(-20 * time.Second),
		},
		{
			name:           "Invalid export timestamp",
			receiveTs:      baseTs,
			inputTraces:    withSpanExportTs(badExportTs, simpleTraces(2*time.Second)),
			expectedTraces: withSpanExportTs(badExportTs, simpleTraces(2 * time.Second)),
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Threshold = time.Second * 5

	tp, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{Logger: zap.NewNop()}, oCfg, consumertest.NewTracesNop())
	require.Nil(t, err)
	require.NotNil(t, tp)
	for _, tc := range testCases {
		runIndividualTestCase(t, tc, tp)
	}
}

func prepareTd(td pdata.Traces) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		rs.Resource().Attributes().Sort()
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			spans := ilss.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				spans.At(k).Attributes().Sort()
			}
		}
	}
}

func withSpanExportTs(exportTs time.Time, td pdata.Traces) pdata.Traces {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			spans := ilss.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				spans.At(k).Attributes().UpsertInt(AttributeSumoTelemetryExportTS, exportTs.UnixNano()/1_000_000)
			}
		}
	}
	return td
}

func simpleTraces(tsDelta time.Duration) pdata.Traces {
	traceID := pdata.NewTraceID([16]byte{1, 2, 3, 4})
	traces := pdata.NewTraces()
	traces.ResourceSpans().Resize(1)
	rs := traces.ResourceSpans().At(0)
	rs.InstrumentationLibrarySpans().Resize(1)
	ils := rs.InstrumentationLibrarySpans().At(0)
	ils.Spans().Resize(1)
	span := ils.Spans().At(0)
	span.SetTraceID(traceID)
	startTs := baseTs.Add(-1 * time.Second)
	endTs := startTs.Add(1 * time.Second)
	span.SetStartTime(pdata.TimeToUnixNano(startTs.Add(tsDelta)))
	span.SetEndTime(pdata.TimeToUnixNano(endTs.Add(tsDelta)))
	span.Attributes().UpsertString("foo", "bar")
	return traces
}
