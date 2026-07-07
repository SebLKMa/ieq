package tasks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	db "github.com/seblkma/ieq/db/postgres"
	intf "github.com/seblkma/ieq/interfaces"
	mdl "github.com/seblkma/ieq/models"
	rate "github.com/seblkma/ieq/ratings"
	awair "github.com/seblkma/ieq/sensors/awair"
	uhoo "github.com/seblkma/ieq/sensors/uhoo"
)

// newDevice constructs the vendor device named in the config.
func (task *ScoringTask) newDevice() (intf.Device, error) {
	switch task.Cfg.VENDOR.Name {
	case "awair":
		return &awair.SensorInfo{Token: task.Cfg.VENDOR.Token, Org: task.Cfg.VENDOR.Org}, nil
	case "uhoo":
		return &uhoo.SensorInfo{Token: task.Cfg.VENDOR.Token, Org: task.Cfg.VENDOR.Org}, nil
	}
	return nil, fmt.Errorf("unknown vendor name %q in config", task.Cfg.VENDOR.Name)
}

// Execute implements interface Executable.Execute()
// It reads the device metrics, computes and stores the scores at the
// configured interval until ctx is cancelled. Scoring starts at the next
// 5 minute boundary of the hour, e.g. :05, :10, :15...
func (task *ScoringTask) Execute(ctx context.Context) error {
	if !task.Initialized {
		return errors.New("scoring task not properly initialized")
	}

	device, err := task.newDevice()
	if err != nil {
		return err
	}

	displayID := task.Cfg.VENDOR.DeviceDisplayID
	interval := time.Duration(task.Cfg.TASK.Minutes) * time.Minute

	// wait until the next 5 minute boundary of the hour
	next := time.Now().Truncate(5 * time.Minute).Add(5 * time.Minute)
	log.Printf("Waiting until %s to start %s task...", next.Format("02/01/2006 15:04:05"), displayID)
	select {
	case <-time.After(time.Until(next)):
	case <-ctx.Done():
		return ctx.Err()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		task.runOnce(ctx, device)
		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Printf("Stopping %s task: %v", displayID, ctx.Err())
			return ctx.Err()
		}
	}
}

// runOnce performs a single fetch-score-store cycle for the device.
// Failures are logged and skipped; the next tick retries.
func (task *ScoringTask) runOnce(ctx context.Context, device intf.Device) {
	vendorDevID := task.Cfg.VENDOR.DeviceID

	devInfo, infoErr := device.GetDeviceInfo(ctx, vendorDevID)
	log.Printf("Executing for %s at %s", devInfo.DeviceID, time.Now().Format("2006/01/02 15:04:05"))

	// record the device status, including error status
	if err := db.CreateDeviceStatus(ctx, devInfo); err != nil {
		log.Println(err)
		return
	}
	if infoErr != nil {
		log.Println(infoErr)
		return
	}
	// no need to continue if device is not connected
	if devInfo.Status == 0 {
		return
	}

	metrics, err := device.GetLatestMetrics(ctx, vendorDevID)
	if err != nil {
		log.Println(err)
		return
	}
	if metrics.Empty {
		log.Printf("%s returned no metrics", devInfo.DeviceID)
		return
	}

	task.scoreThem(ctx, devInfo, metrics)
}

// addScore computes the score for one metric value, records it on the rating
// and stores the raw value and score in the destination records.
// Failures are logged; the metric is then left out of the rating instead of
// polluting it with a fake zero score.
func addScore(r *rate.Rating, sc intf.Scorer, name string, value float64, rawDest, scoreDest *float64) {
	score, err := rate.ComputeScore(sc, value)
	if err != nil {
		log.Println(err)
		return
	}
	if err := r.AddIndex(name, score); err != nil {
		log.Println(err)
		return
	}
	log.Printf("%s: %g   Score: %g", name, value, score)
	*rawDest = value
	*scoreDest = score
}

// computes the metrics scores and store them in database for the device.
func (task *ScoringTask) scoreThem(ctx context.Context, devInfo mdl.DeviceInfo, metrics mdl.Metrics) {

	// set up the weightings
	thermalRating := rate.Rating{}
	thermalRating.Setup("Thermal", task.Cfg.WEIGHTINGS.Thermal)

	iaqRating := rate.Rating{}
	iaqRating.Setup("IAQ", task.Cfg.WEIGHTINGS.IAQ)

	lightingRating := rate.Rating{}
	lightingRating.Setup("Lighting", task.Cfg.WEIGHTINGS.Lighting)

	noiseRating := rate.Rating{}
	noiseRating.Setup("Noise", task.Cfg.WEIGHTINGS.Noise)

	// start computing the metrics scores for the device
	dbmetrics := mdl.Metrics{DeviceID: devInfo.DeviceID}
	dbmetricscore := mdl.MetricScore{DeviceID: devInfo.DeviceID}
	dbieqscore := mdl.IeqScore{DeviceID: devInfo.DeviceID}

	addScore(&thermalRating, task.TemperatureFormula, "Temperature", metrics.Temperature,
		&dbmetrics.Temperature, &dbmetricscore.Temperature)
	addScore(&thermalRating, task.HumidityFormula, "Humidity", metrics.Humidity,
		&dbmetrics.Humidity, &dbmetricscore.Humidity)
	addScore(&iaqRating, task.Co2Formula, "CO2", metrics.CO2,
		&dbmetrics.CO2, &dbmetricscore.CO2)
	addScore(&iaqRating, task.VocFormula, "VOC", metrics.VOC,
		&dbmetrics.VOC, &dbmetricscore.VOC)
	addScore(&iaqRating, task.Pm25Formula, "PM25", metrics.PM25,
		&dbmetrics.PM25, &dbmetricscore.PM25)
	// include lighting score if required
	if lightingRating.Weighting() > 0 {
		addScore(&lightingRating, task.LightingFormula, "Lighting", metrics.Lighting,
			&dbmetrics.Lighting, &dbmetricscore.Lighting)
	}
	// include noise score if required
	if noiseRating.Weighting() > 0 {
		addScore(&noiseRating, task.NoiseFormula, "Noise", metrics.Noise,
			&dbmetrics.Noise, &dbmetricscore.Noise)
	}

	// compute ratings for IEQ components
	thermalRating.SetRating()
	iaqRating.SetRating()
	if lightingRating.Weighting() > 0 {
		lightingRating.SetRating()
	}
	if noiseRating.Weighting() > 0 {
		noiseRating.SetRating()
	}

	// compute IEQ overall rating
	ieqRating := rate.IEQRating{}
	ieqRating.Setup("Overall IEQ", 1.0)
	components := []*rate.Rating{&thermalRating, &iaqRating}
	if lightingRating.Weighting() > 0 {
		components = append(components, &lightingRating)
	}
	if noiseRating.Weighting() > 0 {
		components = append(components, &noiseRating)
	}
	for _, c := range components {
		if err := ieqRating.AddIndex(c.Name(), c.Rate()); err != nil {
			log.Println(err)
		}
	}
	ieqRating.SetRating()

	// start storing to database
	dbieqscore.Scheme = task.Cfg.WEIGHTINGS.Scheme
	dbieqscore.Thermal = thermalRating.Rate()
	dbieqscore.ThermalWeighting = thermalRating.Weighting()
	dbieqscore.IAQ = iaqRating.Rate()
	dbieqscore.IAQWeighting = iaqRating.Weighting()
	if lightingRating.Weighting() > 0 {
		dbieqscore.Lighting = lightingRating.Rate()
		dbieqscore.LightingWeighting = lightingRating.Weighting()
	}
	if noiseRating.Weighting() > 0 {
		dbieqscore.Noise = noiseRating.Rate()
		dbieqscore.NoiseWeighting = noiseRating.Weighting()
	}
	dbieqscore.Overall = ieqRating.Rate()

	for _, i := range ieqRating.Indices() {
		log.Printf("%v ", i)
	}
	log.Printf("%s Rating: %g", ieqRating.Name(), ieqRating.Rate())

	// commit to database
	if err := db.CreateMetric(ctx, dbmetrics); err != nil {
		log.Println(err)
	}
	if err := db.CreateMetricScore(ctx, dbmetricscore); err != nil {
		log.Println(err)
	}
	if err := db.CreateIeqScore(ctx, dbieqscore); err != nil {
		log.Println(err)
	}
}
