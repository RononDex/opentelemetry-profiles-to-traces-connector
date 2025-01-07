package internal

import "go.opentelemetry.io/collector/pdata/pcommon"

type SampleLocation struct {
	Label        string
	DurationInNs int64
	Level        int64
	Attributes   pcommon.Map
}
