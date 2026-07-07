package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mdl "github.com/seblkma/ieq/models"
)

// CreateDeviceStatus creates new record in database
func CreateDeviceStatus(ctx context.Context, data mdl.DeviceInfo) error {
	db, err := getDB()
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx,
		"INSERT INTO devicestatus(device_id, created_on, status, status_desc) VALUES($1, $2, $3, $4)",
		data.DeviceID, time.Now(), data.Status, data.StatusDescription)
	if err != nil {
		return fmt.Errorf("create device status: %w", err)
	}
	return nil
}

// ReadLastDeviceStatus returns the last record
func ReadLastDeviceStatus(ctx context.Context, deviceID string) (mdl.DeviceInfo, error) {
	data := mdl.DeviceInfo{}
	db, err := getDB()
	if err != nil {
		return data, err
	}

	stmt := "SELECT device_id, created_on, status, status_desc FROM devicestatus WHERE device_id = $1 ORDER BY rowid DESC LIMIT 1"
	row := db.QueryRowContext(ctx, stmt, deviceID)
	err = row.Scan(&data.DeviceID, &data.CreatedOn, &data.Status, &data.StatusDescription)
	if errors.Is(err, sql.ErrNoRows) {
		return data, fmt.Errorf("devicestatus for device %s: %w", deviceID, ErrNoRecord)
	}
	if err != nil {
		return data, fmt.Errorf("scan devicestatus: %w", err)
	}
	return data, nil
}
