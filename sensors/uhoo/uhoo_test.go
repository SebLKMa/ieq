package uhoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLatestMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if got := r.PostForm.Get("username"); got != "user@example.com" {
			t.Errorf("username = %q, want user@example.com", got)
		}
		if got := r.PostForm.Get("password"); got != "test&token" {
			t.Errorf("password = %q, want test&token (url encoding broken?)", got)
		}
		if got := r.PostForm.Get("serialNumber"); got != "serial123" {
			t.Errorf("serialNumber = %q, want serial123", got)
		}
		w.Write([]byte(`{
			"Temperature": "24.5",
			"Relative Humidity": "60.1",
			"CO2": "410",
			"TVOC": "88",
			"PM2.5": "9"
		}`))
	}))
	defer srv.Close()

	oldBase := uhooBaseURL
	uhooBaseURL = srv.URL
	defer func() { uhooBaseURL = oldBase }()

	// the token contains a form metacharacter on purpose: it must be encoded
	sensor := SensorInfo{Token: "test&token", Org: "user@example.com"}
	metrics, err := sensor.GetLatestMetrics(context.Background(), "serial123")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Empty {
		t.Error("metrics unexpectedly empty")
	}
	if metrics.Temperature != 24.5 {
		t.Errorf("Temperature = %g, want 24.5", metrics.Temperature)
	}
	if metrics.Humidity != 60.1 {
		t.Errorf("Humidity = %g, want 60.1", metrics.Humidity)
	}
	if metrics.VOC != 88 {
		t.Errorf("VOC = %g, want 88", metrics.VOC)
	}
}

// Regression test: a non-2xx response must surface as an error instead of the
// error page being fed to the JSON decoder.
func TestGetLatestMetricsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldBase := uhooBaseURL
	uhooBaseURL = srv.URL
	defer func() { uhooBaseURL = oldBase }()

	sensor := SensorInfo{Token: "t", Org: "o"}
	if _, err := sensor.GetLatestMetrics(context.Background(), "serial123"); err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestGetDeviceInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"deviceName": "uhoo_device", "serialNumber": "serial123", "company": "ACME", "macAddress": "70886b0000"}]`))
	}))
	defer srv.Close()

	oldBase := uhooBaseURL
	uhooBaseURL = srv.URL
	defer func() { uhooBaseURL = oldBase }()

	sensor := SensorInfo{Token: "t", Org: "o"}
	info, err := sensor.GetDeviceInfo(context.Background(), "serial123")
	if err != nil {
		t.Fatal(err)
	}
	if info.DeviceID != "uhoo_device" {
		t.Errorf("DeviceID = %q, want uhoo_device", info.DeviceID)
	}
	if info.SerialNumber != "serial123" {
		t.Errorf("SerialNumber = %q, want serial123", info.SerialNumber)
	}
	if info.Status != 1 {
		t.Errorf("Status = %d, want 1", info.Status)
	}
}
