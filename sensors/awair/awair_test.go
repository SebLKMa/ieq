package awair

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const airDataJSON = `{
	"data": [{
		"timestamp": "2021-02-23T02:55:09.000Z",
		"score": 93,
		"sensors": [
			{"comp": "temp", "value": 29.24},
			{"comp": "humid", "value": 59.81},
			{"comp": "co2", "value": 410},
			{"comp": "voc", "value": 88},
			{"comp": "pm25", "value": 9},
			{"comp": "lux", "value": 144.9},
			{"comp": "spl_a", "value": 54.7}
		]
	}]
}`

func TestGetLatestMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Api-Key"); got != "test-token" {
			t.Errorf("X-Api-Key = %q, want test-token", got)
		}
		w.Write([]byte(airDataJSON))
	}))
	defer srv.Close()

	oldBase := awairBaseURL
	awairBaseURL = srv.URL
	defer func() { awairBaseURL = oldBase }()

	sensor := SensorInfo{Token: "test-token", Org: "3332"}
	metrics, err := sensor.GetLatestMetrics(context.Background(), "18453")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Empty {
		t.Error("metrics unexpectedly empty")
	}
	if metrics.Temperature != 29.24 {
		t.Errorf("Temperature = %g, want 29.24", metrics.Temperature)
	}
	if metrics.CO2 != 410 {
		t.Errorf("CO2 = %g, want 410", metrics.CO2)
	}
	if metrics.Noise != 54.7 {
		t.Errorf("Noise = %g, want 54.7", metrics.Noise)
	}
}

// Regression test: a non-2xx response must surface as an error instead of the
// error page being fed to the JSON decoder.
func TestGetLatestMetricsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	oldBase := awairBaseURL
	awairBaseURL = srv.URL
	defer func() { awairBaseURL = oldBase }()

	sensor := SensorInfo{Token: "bad-token", Org: "3332"}
	if _, err := sensor.GetLatestMetrics(context.Background(), "18453"); err == nil {
		t.Error("expected error for 401 response")
	}
}

func TestGetDeviceInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"uuid": "awair-omni_18453", "mac_address": "70886b0000", "display_name": "Meeting Room", "org_id": 3332, "connected": true}`))
	}))
	defer srv.Close()

	oldBase := awairBaseURL
	awairBaseURL = srv.URL
	defer func() { awairBaseURL = oldBase }()

	sensor := SensorInfo{Token: "test-token", Org: "3332"}
	info, err := sensor.GetDeviceInfo(context.Background(), "18453")
	if err != nil {
		t.Fatal(err)
	}
	if info.DeviceID != "awair-omni_18453" {
		t.Errorf("DeviceID = %q", info.DeviceID)
	}
	if info.Status != 1 || info.StatusDescription != "connected" {
		t.Errorf("Status = %d (%s), want 1 (connected)", info.Status, info.StatusDescription)
	}
	if info.Org != "3332" {
		t.Errorf("Org = %q, want 3332", info.Org)
	}
}
