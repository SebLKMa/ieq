package awair

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SensorInfo properties
type SensorInfo struct {
	Token string
	Org   string
}

// awairBaseURL is a variable so tests can point it at a fake server.
var awairBaseURL = "https://developer-apis.awair.is/v1"

// httpClient is shared by all requests; http.DefaultClient has no timeout.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// get performs an authenticated GET against the awair cloud API and returns
// the response body. Non-2xx responses are returned as errors.
func (sensor *SensorInfo) get(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", sensor.Token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("awair API %s: %s", url, resp.Status)
	}
	return string(body), nil
}
