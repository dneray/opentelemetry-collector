package sampling

import (
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/open-telemetry/opentelemetry-collector/internal"
	"go.uber.org/zap"
	"time"
)

type durationFilter struct {
	minValue, maxValue *time.Duration
	logger             *zap.Logger
}

func (d durationFilter) OnLateArrivingSpans(earlyDecision Decision, spans []*tracepb.Span) error {
	d.logger.Debug("Triggering action for late arriving spans in numeric-attribute filter")
	return nil
}

func (d durationFilter) Evaluate(traceID []byte, trace *TraceData) (Decision, error) {
	d.logger.Debug("Evaluating spans in numeric-attribute filter")
	trace.Lock()
	batches := trace.ReceivedBatches
	trace.Unlock()
	for _, batch := range batches {
		for _, span := range batch.Spans {
			if span == nil{
				continue
			}
			if len(span.ParentSpanId) != 0 {
				// only care about the root span
				continue
			}
			actualDuration := internal.TimestampToTime(span.EndTime).Sub(internal.TimestampToTime(span.StartTime))

			if (d.minValue != nil && actualDuration < *d.minValue) || (d.maxValue != nil && actualDuration > *d.maxValue) {
				return NotSampled, nil
			}
			return Sampled, nil
		}
	}

	return NotSampled, nil
}

func (d durationFilter) OnDroppedSpans(traceID []byte, trace *TraceData) (Decision, error) {
	d.logger.Debug("Triggering action for dropped spans in duration filter")
	return NotSampled, nil
}

func NewDurationFilter( logger *zap.Logger, minValueMs *int64, maxValueMs *int64) PolicyEvaluator {
	var minValue *time.Duration
	if minValueMs != nil {
		minValue_dur := time.Duration(*minValueMs) * time.Millisecond
		minValue = &minValue_dur
	}
	var maxValue *time.Duration
	if maxValueMs != nil {
		maxValue_dur := time.Duration(*maxValueMs) * time.Millisecond
		maxValue = &maxValue_dur
	}

	return &durationFilter{logger: logger, minValue: minValue, maxValue: maxValue}
}
