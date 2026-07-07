package awair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mdl "github.com/seblkma/ieq/models"
)

// deviceState matches the awair device JSON
type deviceState struct {
	UUID        string `json:"uuid"`
	MacAddress  string `json:"mac_address"`
	DisplayName string `json:"display_name"`
	OrgID       any    `json:"org_id"` // vendor may send a string or a number
	Connected   bool   `json:"connected"`
}

// GetState implements device interface Device.GetState()
// Uses sensor cloud API to get device information.
func (sensor *SensorInfo) GetState(ctx context.Context, id string) (result string, err error) {
	apiurl := awairBaseURL + "/orgs/" + sensor.Org + "/devices/awair-omni/" + id
	return sensor.get(ctx, apiurl)
}

// GetRawMetrics implements device interface Device.GetRawMetrics()
// Uses sensor cloud API to get metrics values.
func (sensor *SensorInfo) GetRawMetrics(ctx context.Context, id string) (result string, err error) {
	apiurl := awairBaseURL + "/orgs/" + sensor.Org + "/devices/awair-omni/" + id + "/air-data/latest"
	return sensor.get(ctx, apiurl)
}

// GetDeviceInfo returns the current information of the device
func (sensor *SensorInfo) GetDeviceInfo(ctx context.Context, id string) (result mdl.DeviceInfo, err error) {
	result.VendorDeviceID = id
	result.CreatedOn = time.Now()

	jsonData, err := sensor.GetState(ctx, id)
	if err != nil {
		result.Status = 0
		result.StatusDescription = err.Error()
		log.Println(err)
		return result, err
	}

	var state deviceState
	if err = json.Unmarshal([]byte(jsonData), &state); err != nil {
		result.Status = 0
		result.StatusDescription = err.Error()
		log.Println(err)
		return result, err
	}

	result.DeviceID = state.UUID
	result.MacAddress = state.MacAddress
	result.DisplayName = state.DisplayName
	if state.OrgID != nil {
		result.Org = fmt.Sprint(state.OrgID)
	}
	if state.Connected {
		result.Status = 1
		result.StatusDescription = "connected"
	} else {
		result.Status = 0
		result.StatusDescription = "disconnected"
	}

	return result, nil
}
