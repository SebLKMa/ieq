package interfaces

import (
	"context"

	mdl "github.com/seblkma/ieq/models"
)

// Device represents a device containing sensors to provide measurements.
// Authentication should be done by the concrete implementation.
// The context carries cancellation and deadlines for the underlying vendor API calls.
type Device interface {
	GetState(ctx context.Context, id string) (result string, err error)
	GetDeviceInfo(ctx context.Context, id string) (result mdl.DeviceInfo, err error)
	GetRawMetrics(ctx context.Context, id string) (result string, err error)
	GetLatestMetrics(ctx context.Context, deviceID string) (result mdl.Metrics, err error)
}
