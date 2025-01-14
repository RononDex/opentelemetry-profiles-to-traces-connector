package internal

import "go.opentelemetry.io/collector/pdata/pcommon"

type SampleLocation struct {
	Label          string
	DurationInNs   int64
	Level          int64
	Self           int64
	StartTimeStamp pcommon.Timestamp
	EndTimeStamp   pcommon.Timestamp
	Attributes     pcommon.Map
	ParentSpanId   pcommon.SpanID
}
