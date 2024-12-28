package profilestotraces

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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
	for i := 0; i < profiles.ResourceProfiles().Len(); i++ {
		resourceProfile := profiles.ResourceProfiles().At(i)
		newResourceTrace := traces.ResourceSpans().AppendEmpty()

		for j := 0; j < resourceProfile.ScopeProfiles().Len(); j++ {
			scopeProfile := resourceProfile.ScopeProfiles().At(j)

			for k := 0; k < scopeProfile.Profiles().Len(); k++ {
				profile := scopeProfile.Profiles().At(k)

				for l := 0; l < profile.Sample().Len(); l++ {
					sample := profile.Sample().At(l)
					newTraceSpan := newResourceTrace.ScopeSpans().AppendEmpty()

					copyAttributes(sample, profile, newTraceSpan)
				}

			}
		}
	}

	return c.tracesConsumer.ConsumeTraces(ctx, traces)
}

func copyAttributes(sample pprofile.Sample, profile pprofile.Profile, newTraceSpan ptrace.ScopeSpans) {
	for m := 0; m < sample.AttributeIndices().Len(); m++ {
		attributeTableEntry := profile.AttributeTable().At(int(sample.AttributeIndices().At(m)))
		if attributeTableEntry.Key() == "process.executable.path" {
			newTraceSpan.Scope().SetName(attributeTableEntry.Value().Str())
		}
		newAttribute := newTraceSpan.Scope().Attributes().PutEmpty(attributeTableEntry.Key())
		attributeTableEntry.Value().CopyTo(newAttribute)
	}
}
