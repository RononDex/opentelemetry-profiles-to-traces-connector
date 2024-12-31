package profilestotraces

import (
	"context"
	"crypto/rand"
	"strconv"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

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
				profile := scopeProfile.Profiles().At(k)
				profileSpan := scopeSpans.Spans().AppendEmpty()
				profileSpan.SetTraceID(traceId)
				profileSpan.SetName("Profile_" + profile.ProfileID().String())
				profileSpan.SetSpanID(createNewSpanId())
				profileSpan.SetKind(ptrace.SpanKindInternal)
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

					copyAttributes(sample.AttributeIndices(), profile.AttributeTable(), sampleSpan)
					copyLocations(sample, profile, profileSpan, scopeSpans, traceId)
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

func copyLocations(sample pprofile.Sample, profile pprofile.Profile, profileSpan ptrace.Span, scopeSpans ptrace.ScopeSpans, traceId pcommon.TraceID) {
	locationIdxOffset := sample.LocationsStartIndex()
	parentSpanId := profileSpan.SpanID()
	for locationIdx := sample.LocationsLength() - 1; locationIdx >= 0; locationIdx-- {
		location := profile.LocationTable().At(int(locationIdx) + int(locationIdxOffset))

		locationSpan := scopeSpans.Spans().AppendEmpty()
		locationSpan.SetTraceID(traceId)
		locationSpan.SetSpanID(createNewSpanId())
		locationSpan.SetParentSpanID(parentSpanId)
		locationSpan.SetName("Location")

		if location.Line().Len() > 0 {
			line := location.Line().At(0)
			functionName := profile.StringTable().At(int(profile.FunctionTable().At(int(line.FunctionIndex())).NameStrindex()))

			locationSpan.SetName(functionName)
			locationSpan.Attributes().PutInt("location.lineNr", line.Line())
			locationSpan.Attributes().PutInt("location.columnNr", line.Column())
			locationSpan.Attributes().PutStr("location.functionName", functionName)
		}
		if location.HasMappingIndex() {
			mappingIdx := location.MappingIndex()
			mapping := profile.MappingTable().At(int(mappingIdx))

			if mapping.HasFilenames() {
				locationSpan.Attributes().PutStr("mapping.fileName", profile.StringTable().At(int(mapping.FilenameStrindex())))
			}
		}

		copyAttributes(location.AttributeIndices(), profile.AttributeTable(), locationSpan)

		parentSpanId = locationSpan.SpanID()
	}
}

func copyAttributes(attributeIndices pcommon.Int32Slice, attributeTable pprofile.AttributeTableSlice, targetSpan ptrace.Span) {
	for m := 0; m < attributeIndices.Len(); m++ {
		attributeTableEntry := attributeTable.At(int(attributeIndices.At(m)))
		newAttribute := targetSpan.Attributes().PutEmpty(attributeTableEntry.Key())
		attributeTableEntry.Value().CopyTo(newAttribute)
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
