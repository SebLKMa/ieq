package uhoo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SensorInfo properties
type SensorInfo struct {
	Token string
	Org   string
}

// uhooBaseURL is a variable so tests can point it at a fake server.
var uhooBaseURL = "https://api.uhooinc.com/v1"

// httpClient is shared by all requests; http.DefaultClient has no timeout.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// postForm performs an authenticated form POST against the uHoo cloud API and
// returns the response body. Non-2xx responses are returned as errors.
func (sensor *SensorInfo) postForm(ctx context.Context, apiurl string, extra url.Values) (string, error) {
	form := url.Values{}
	form.Set("username", sensor.Org)
	form.Set("password", sensor.Token)
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiurl, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", fmt.Errorf("uhoo API %s: %s", apiurl, resp.Status)
	}
	return string(body), nil
}
