package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mdl "github.com/seblkma/ieq/models"
)

// ErrNoRecord is returned when a query matches no rows, so callers can
// distinguish "no data yet" from a real failure.
var ErrNoRecord = errors.New("no record found")

const (
	metricsColumns  = "device_id, created_on, temperature, humidity, co2, voc, pm25, lighting, noise"
	ieqScoreColumns = "device_id, created_on, scheme, thermal, iaq, lighting, noise, overall, thermal_weighting, iaq_weighting, lighting_weighting, noise_weighting"
)

// CreateMetric creates metrics record in database
func CreateMetric(ctx context.Context, data mdl.Metrics) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	sqlstmt := "INSERT INTO metrics(" + metricsColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err = db.ExecContext(ctx, sqlstmt,
		data.DeviceID,
		time.Now(),
		data.Temperature,
		data.Humidity,
		data.CO2,
		data.VOC,
		data.PM25,
		data.Lighting,
		data.Noise)
	if err != nil {
		return fmt.Errorf("create metrics: %w", err)
	}
	return nil
}

// CreateMetricScore creates metricscores record in database
func CreateMetricScore(ctx context.Context, data mdl.MetricScore) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	sqlstmt := "INSERT INTO metricscores(" + metricsColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)"
	_, err = db.ExecContext(ctx, sqlstmt,
		data.DeviceID,
		time.Now(),
		data.Temperature,
		data.Humidity,
		data.CO2,
		data.VOC,
		data.PM25,
		data.Lighting,
		data.Noise)
	if err != nil {
		return fmt.Errorf("create metricscores: %w", err)
	}
	return nil
}

// CreateIeqScore creates ieqscores record in database
func CreateIeqScore(ctx context.Context, data mdl.IeqScore) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	sqlstmt := "INSERT INTO ieqscores(" + ieqScoreColumns + ") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)"
	_, err = db.ExecContext(ctx, sqlstmt,
		data.DeviceID,
		time.Now(),
		data.Scheme,
		data.Thermal,
		data.IAQ,
		data.Lighting,
		data.Noise,
		data.Overall,
		data.ThermalWeighting,
		data.IAQWeighting,
		data.LightingWeighting,
		data.NoiseWeighting)
	if err != nil {
		return fmt.Errorf("create ieqscores: %w", err)
	}
	return nil
}

// scanMetrics scans one metrics/metricscores row into the destination fields.
func scanMetricsRow(row interface{ Scan(dest ...any) error }, deviceID, table string, dest ...any) error {
	err := row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s for device %s: %w", table, deviceID, ErrNoRecord)
	}
	if err != nil {
		return fmt.Errorf("scan %s: %w", table, err)
	}
	return nil
}

// ReadLatestMetrics returns the latest metrics record
func ReadLatestMetrics(ctx context.Context, deviceID string) (mdl.Metrics, error) {
	data := mdl.Metrics{}
	db, err := getDB()
	if err != nil {
		return data, err
	}

	stmt := "SELECT " + metricsColumns + " FROM metrics WHERE device_id = $1 ORDER BY rowid DESC LIMIT 1"
	row := db.QueryRowContext(ctx, stmt, deviceID)
	err = scanMetricsRow(row, deviceID, "metrics",
		&data.DeviceID, &data.CreatedOn, &data.Temperature, &data.Humidity,
		&data.CO2, &data.VOC, &data.PM25, &data.Lighting, &data.Noise)
	return data, err
}

// ReadLatestMetricScores returns the latest metricscores record
func ReadLatestMetricScores(ctx context.Context, deviceID string) (mdl.MetricScore, error) {
	data := mdl.MetricScore{}
	db, err := getDB()
	if err != nil {
		return data, err
	}

	stmt := "SELECT " + metricsColumns + " FROM metricscores WHERE device_id = $1 ORDER BY rowid DESC LIMIT 1"
	row := db.QueryRowContext(ctx, stmt, deviceID)
	err = scanMetricsRow(row, deviceID, "metricscores",
		&data.DeviceID, &data.CreatedOn, &data.Temperature, &data.Humidity,
		&data.CO2, &data.VOC, &data.PM25, &data.Lighting, &data.Noise)
	return data, err
}

// ReadLatestIeqScores returns the latest ieqscores record
func ReadLatestIeqScores(ctx context.Context, deviceID string) (mdl.IeqScore, error) {
	data := mdl.IeqScore{}
	db, err := getDB()
	if err != nil {
		return data, err
	}

	stmt := "SELECT " + ieqScoreColumns + " FROM ieqscores WHERE device_id = $1 ORDER BY rowid DESC LIMIT 1"
	row := db.QueryRowContext(ctx, stmt, deviceID)
	err = scanMetricsRow(row, deviceID, "ieqscores",
		&data.DeviceID, &data.CreatedOn, &data.Scheme, &data.Thermal, &data.IAQ,
		&data.Lighting, &data.Noise, &data.Overall, &data.ThermalWeighting,
		&data.IAQWeighting, &data.LightingWeighting, &data.NoiseWeighting)
	return data, err
}

// ReadMetrics returns up to count of the most recent metrics records
func ReadMetrics(ctx context.Context, deviceID string, count int) ([]mdl.Metrics, error) {
	results := []mdl.Metrics{}
	db, err := getDB()
	if err != nil {
		return results, err
	}

	stmt := "SELECT " + metricsColumns + " FROM metrics WHERE device_id = $1 ORDER BY rowid DESC LIMIT $2"
	rows, err := db.QueryContext(ctx, stmt, deviceID, count)
	if err != nil {
		return results, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item := mdl.Metrics{}
		err = rows.Scan(&item.DeviceID, &item.CreatedOn, &item.Temperature, &item.Humidity,
			&item.CO2, &item.VOC, &item.PM25, &item.Lighting, &item.Noise)
		if err != nil {
			return results, fmt.Errorf("scan metrics: %w", err)
		}
		results = append(results, item)
	}
	if err = rows.Err(); err != nil {
		return results, fmt.Errorf("iterate metrics: %w", err)
	}

	if len(results) == 0 {
		return results, fmt.Errorf("metrics for device %s: %w", deviceID, ErrNoRecord)
	}

	return results, nil
}
