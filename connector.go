package profilestotraces

import (
	"context"
	"crypto/rand"
	"strconv"
	"time"

	"github.com/RononDex/profilestotracesconnector/internal"
	"github.com/RononDex/profilestotracesconnector/tree"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Profile signal documentation:
// 	    https://github.com/open-telemetry/opentelemetry-specification/blob/main/oteps/profiles/0239-profiles-data-model.md#message-profile
// Grafana data needed for flamegraph:
//      https://grafana.com/docs/grafana/latest/panels-visualizations/visualizations/flame-graph/
// TODO:
//   - Create graph structure
//   - Edge cases: Location without sample value --> take value from lower location
//   - Create one Span per location

// schema for connector
type connectorImp struct {
	config         Config
	tracesConsumer consumer.Traces
	logger         *zap.Logger
	// Include these parameters if a specific implementation for the Start and Shutdown function are not needed
	component.StartFunc
	component.ShutdownFunc
}

// newConnector is a function to create a new connector
func newConnector(logger *zap.Logger, config component.Config) (*connectorImp, error) {
	logger.Info("Building exampleconnector connector")
	cfg := config.(*Config)

	return &connectorImp{
		config: *cfg,
		logger: logger,
	}, nil
}

// Capabilities implements the consumer interface.
func (c *connectorImp) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces method is called for each instance of a trace sent to the connector
func (c *connectorImp) ConsumeProfiles(ctx context.Context, profiles pprofile.Profiles) error {
	// loop through the levels of spans of the one trace consumed
	traces := ptrace.NewTraces()
	traceId := createNewTraceId()

	for i := 0; i < profiles.ResourceProfiles().Len(); i++ {
		resourceProfile := profiles.ResourceProfiles().At(i)
		newResourceTrace := traces.ResourceSpans().AppendEmpty()

		for j := 0; j < resourceProfile.ScopeProfiles().Len(); j++ {
			scopeProfile := resourceProfile.ScopeProfiles().At(j)
			scopeSpans := newResourceTrace.ScopeSpans().AppendEmpty()
			scopeProfile.Scope().Attributes().CopyTo(scopeSpans.Scope().Attributes())

			for k := 0; k < scopeProfile.Profiles().Len(); k++ {
				profileTree := tree.Tree[internal.SampleLocation]{}
				profileTree.RootNode = tree.Node[internal.SampleLocation]{}
				profile := scopeProfile.Profiles().At(k)
				profileSpan := scopeSpans.Spans().AppendEmpty()
				profileSpan.SetTraceID(traceId)
				profileSpan.SetName("Profile_" + profile.ProfileID().String())
				profileSpan.SetSpanID(createNewSpanId())
				profileSpan.SetKind(ptrace.SpanKindInternal)
				profileSpan.SetStartTimestamp(profile.StartTime())
				profile.Attributes().CopyTo(profileSpan.Attributes())
				setProfileAttributes(profile, profileSpan)

				for l := 0; l < profile.Sample().Len(); l++ {
					sample := profile.Sample().At(l)
					sampleSpan := scopeSpans.Spans().AppendEmpty()
					sampleSpan.SetTraceID(traceId)
					sampleSpan.SetSpanID(createNewSpanId())
					sampleSpan.SetName("Sample")
					sampleSpan.SetKind(ptrace.SpanKindInternal)
					sampleSpan.SetParentSpanID(profileSpan.SpanID())
					sampleSpan.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Unix(0, int64(sample.TimestampsUnixNano().At(0)))))

					// Grafana compliant attributes
					sampleSpan.Attributes().PutInt("value", sample.Value().At(0))
					sampleSpan.Attributes().PutInt("level", int64(sample.LocationsLength()))
					// sampleSpan.Attributes().PutStr("label")
					copyLocationsToTree(sample, profile, profileSpan, scopeSpans, traceId, profileTree)
				}
			}
		}
	}

	return c.tracesConsumer.ConsumeTraces(ctx, traces)
}

func setProfileAttributes(profile pprofile.Profile, profileSpan ptrace.Span) {
	profileSpan.Attributes().PutStr("profile.time", profile.Time().String())
	profileSpan.Attributes().PutInt("profile.period", profile.Period())
	profileSpan.Attributes().PutStr("profile.duration", profile.Duration().String())
	profileSpan.Attributes().PutStr("profile.startTime", profile.StartTime().String())
	profileSpan.Attributes().PutStr("profile.periodType", profile.StringTable().At(int(profile.PeriodType().TypeStrindex())))

	for i := 0; i < profile.CommentStrindices().Len(); i++ {
		commentIdx := profile.CommentStrindices().At(i)
		comment := profile.StringTable().At(int(commentIdx))
		profileSpan.Attributes().PutStr("Comment"+strconv.Itoa(i+1), comment)
	}
}

func copyLocationsToTree(sample pprofile.Sample, profile pprofile.Profile, profileSpan ptrace.Span, scopeSpans ptrace.ScopeSpans, traceId pcommon.TraceID, sampleTree tree.Tree[internal.SampleLocation]) {
	locationIdxOffset := sample.LocationsStartIndex()
	parentSpanId := profileSpan.SpanID()
	currentNode := sampleTree.RootNode

	for locationIdx := 0; locationIdx < int(sample.LocationsLength()); locationIdx++ {
		location := profile.LocationTable().At(int(locationIdx) + int(locationIdxOffset))
		functionName := strconv.FormatUint(location.Address(), 10)
		if location.Line().Len() > 0 {
			functionName = profile.StringTable().At(int(profile.FunctionTable().At(int(location.Line().At(0).FunctionIndex())).NameStrindex()))
		}

		subNode := findSubNodeByLabel(currentNode, functionName)
		if subNode == nil {
			newNode := tree.Node[internal.SampleLocation]{}
			currentNode.SubNodes = append(currentNode.SubNodes, newNode)
			newNode.Value = internal.SampleLocation{}
			newNode.Value.Label = functionName
			newNode.Value.Level = int64(locationIdx)
			newNode.Value.Attributes = pcommon.Map{}
			newNode.Value.DurationInNs = 0

			subNode = &newNode
		}

		if locationIdx == int(sample.LocationsLength())-1 {
			subNode.Value.DurationInNs = sample.Value().At(0)
			copyAttributes(sample.AttributeIndices(), profile.AttributeTable(), subNode.Value.Attributes)
		}

		currentNode = *subNode
	}
}

func findSubNodeByLabel(sampleNode tree.Node[internal.SampleLocation], label string) *tree.Node[internal.SampleLocation] {
	for nodeIdx := 0; nodeIdx < len(sampleNode.SubNodes); nodeIdx++ {
		if sampleNode.SubNodes[nodeIdx].Value.Label == label {
			return &sampleNode.SubNodes[nodeIdx]
		}
	}

	return nil
}

func copyAttributes(attributeIndices pcommon.Int32Slice, attributeTable pprofile.AttributeTableSlice, targetAttributes pcommon.Map) {
	for m := 0; m < attributeIndices.Len(); m++ {
		attributeTableEntry := attributeTable.At(int(attributeIndices.At(m)))
		_, exists := targetAttributes.Get(attributeTableEntry.Key())
		if !exists {
			newAttribute := targetAttributes.PutEmpty(attributeTableEntry.Key())
			attributeTableEntry.Value().CopyTo(newAttribute)
		}
	}
}

func createNewTraceId() pcommon.TraceID {
	traceIdBytes := make([]byte, 16)
	_, err := rand.Read(traceIdBytes)
	if err != nil {
		panic("Error while creating new trace ID: " + err.Error())
	}
	traceId := pcommon.TraceID(traceIdBytes)

	return traceId
}

func createNewSpanId() pcommon.SpanID {
	spanIdBytes := make([]byte, 8)
	_, err := rand.Read(spanIdBytes)
	if err != nil {
		panic("Error while creating new span ID: " + err.Error())
	}
	spanId := pcommon.SpanID(spanIdBytes)

	return spanId
}
