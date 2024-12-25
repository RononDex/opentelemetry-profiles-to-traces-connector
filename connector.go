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
			newTraceSpan := newResourceTrace.ScopeSpans().AppendEmpty()

			newTraceSpan.Scope().SetName(scopeProfile.Scope().Name())
			scopeProfile.Scope().Attributes().CopyTo(newTraceSpan.Scope().Attributes())

			for k := 0; k < scopeProfile.Profiles().Len(); k++ {
				profile := scopeProfile.Profiles().At(k)
				newSpan := newTraceSpan.Scope().Attributes().PutEmpty(string(profile.ProfileID().String()))
				newSpan.SetStr("Test")
			}
		}
	}

	return c.tracesConsumer.ConsumeTraces(ctx, traces)
}
