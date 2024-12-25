package profilestotraces

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
)

var (
	defaultVal = "request.n"
	// this is the name used to refer to the connector in the config.yaml
	typeStr       = "profilestotraces"
	connectorType = component.MustNewType("profilestotraces")
)

// NewFactory creates a factory for example connector.
func NewFactory() xconnector.Factory {
	// OpenTelemetry connector factory to make a factory for connectors

	return xconnector.NewFactory(
		connectorType,
		createDefaultConfig,
		xconnector.WithProfilesToTraces(createProfilesToTraces, component.StabilityLevelAlpha))
}

func createDefaultConfig() component.Config {
	return &Config{
		AttributeName: defaultVal,
	}
}

// createTracesToMetricsConnector defines the consumer type of the connector
// We want to consume traces and export metrics, therefore, define nextConsumer as metrics, since consumer is the next component in the pipeline
func createProfilesToTraces(ctx context.Context, params connector.Settings, cfg component.Config, traces consumer.Traces) (nextConsumer xconnector.Profiles, err error) {
	c, err := newConnector(params.Logger, cfg)
	if err != nil {
		return nil, err
	}
	c.tracesConsumer = traces
	return c, nil
}
