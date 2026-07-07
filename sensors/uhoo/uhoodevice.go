package uhoo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	mdl "github.com/seblkma/ieq/models"
)

// deviceEntry matches one device in the uHoo device list JSON
type deviceEntry struct {
	DeviceName   string `json:"deviceName"`
	SerialNumber string `json:"serialNumber"`
	Company      string `json:"company"`
	MacAddress   string `json:"macAddress"`
}

// latestData matches the uHoo latest data JSON; all values arrive as strings
type latestData struct {
	Temperature string `json:"Temperature"`
	Humidity    string `json:"Relative Humidity"`
	CO2         string `json:"CO2"`
	TVOC        string `json:"TVOC"`
	PM25        string `json:"PM2.5"`
}

// GetState implements device interface Device.GetState()
// Uses sensor cloud API to get device information.
func (sensor *SensorInfo) GetState(ctx context.Context, id string) (result string, err error) {
	return sensor.postForm(ctx, uhooBaseURL+"/getdevicelist", nil)
}

// GetRawMetrics implements device interface Device.GetRawMetrics()
// Uses sensor cloud API to get metrics values.
func (sensor *SensorInfo) GetRawMetrics(ctx context.Context, id string) (result string, err error) {
	return sensor.postForm(ctx, uhooBaseURL+"/getlatestdata", url.Values{"serialNumber": {id}})
}

// parseMetric parses one string metric value into dest and clears the Empty flag.
func parseMetric(name, value string, dest *float64, empty *bool) error {
	if value == "" {
		return nil
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("parse %s %q: %w", name, value, err)
	}
	*dest = v
	*empty = false
	return nil
}

// GetLatestMetrics gets the latest raw metrics from device
func (sensor *SensorInfo) GetLatestMetrics(ctx context.Context, deviceID string) (result mdl.Metrics, err error) {
	result = mdl.Metrics{Empty: true, CreatedOn: time.Now()}

	jsonData, err := sensor.GetRawMetrics(ctx, deviceID)
	if err != nil {
		log.Println(err)
		return result, err
	}

	var data latestData
	if err = json.Unmarshal([]byte(jsonData), &data); err != nil {
		log.Println(err)
		return result, err
	}

	for _, m := range []struct {
		name  string
		value string
		dest  *float64
	}{
		{"Temperature", data.Temperature, &result.Temperature},
		{"Humidity", data.Humidity, &result.Humidity},
		{"CO2", data.CO2, &result.CO2},
		{"VOC", data.TVOC, &result.VOC},
		{"PM25", data.PM25, &result.PM25},
	} {
		if err = parseMetric(m.name, m.value, m.dest, &result.Empty); err != nil {
			return result, err
		}
	}

	return result, nil
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

	var devices []deviceEntry
	if err = json.Unmarshal([]byte(jsonData), &devices); err != nil {
		result.Status = 0
		result.StatusDescription = err.Error()
		log.Println(err)
		return result, err
	}

	// vendor API has no way to tell if device is online or offline
	result.Status = 1

	for _, d := range devices {
		result.DeviceID = d.DeviceName
		result.DisplayName = d.DeviceName
		result.SerialNumber = d.SerialNumber
		result.Org = d.Company
		result.MacAddress = d.MacAddress
	}

	return result, nil
}
