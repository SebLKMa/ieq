package awair

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	mdl "github.com/seblkma/ieq/models"
)

// compValue key value pair to match awair json
type compValue struct {
	Comp  string  `json:"comp"`
	Value float64 `json:"value"`
}

// rawData defines the raw metric data to match awair json
type rawData struct {
	Timestamp time.Time   `json:"timestamp"`
	Score     float64     `json:"score"`
	Sensors   []compValue `json:"sensors"`
}

// rawDataList defines a list of raw data to match awair json
type rawDataList struct {
	Data []rawData `json:"data"`
}

// GetLatestMetrics gets the latest raw metrics from device
func (sensor *SensorInfo) GetLatestMetrics(ctx context.Context, deviceID string) (result mdl.Metrics, err error) {
	result = mdl.Metrics{Empty: true, CreatedOn: time.Now()}

	jsonData, err := sensor.GetRawMetrics(ctx, deviceID)
	if err != nil {
		log.Println(err)
		return result, err
	}

	var rawList rawDataList
	err = json.Unmarshal([]byte(jsonData), &rawList)
	if err != nil {
		log.Println(err)
		return result, err
	}

	if len(rawList.Data) == 0 {
		return result, errors.New("awair data is empty")
	}

	// first element contains the raw metrics
	for _, s := range rawList.Data[0].Sensors {
		switch s.Comp {
		case "temp":
			result.Temperature = s.Value
			result.Empty = false
		case "humid":
			result.Humidity = s.Value
			result.Empty = false
		case "co2":
			result.CO2 = s.Value
			result.Empty = false
		case "voc":
			result.VOC = s.Value
			result.Empty = false
		case "pm25":
			result.PM25 = s.Value
			result.Empty = false
		case "lux":
			result.Lighting = s.Value
			result.Empty = false
		case "spl_a":
			result.Noise = s.Value
			result.Empty = false
		}
	}

	return result, nil
}
