//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func collectMetrics(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	if metricsInstance == nil {
		return errors.New("metrics not initialized")
	}

	err := metricsInstance.promExporter.Collect(ctx, rm)
	if err != nil {
		return fmt.Errorf("collect metrics: %w", err)
	}

	return nil
}
