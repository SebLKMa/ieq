package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	db "github.com/seblkma/ieq/db/postgres"
	"github.com/seblkma/ieq/gotemplates"
	mdl "github.com/seblkma/ieq/models"
)

var templates = template.Must(template.ParseFS(gotemplates.FS,
	"common/customstyle.html",
	"common/customscript.html",
	"common/footer.html",
	"common/metrics.html",
	"common/metricscores.html",
	"common/status.html",
	"ieq/header.html",
	"ieq/ieqscores.html",
	"ieqcharts/chartcss.html",
	"ieqcharts/heartscss.html",
	"ieqcharts/scriptline.html",
	"ieqcharts/scriptdonut.html",
	"ieqcharts/scripthbarstacked.html",
	"ieqcharts/scripthearts.html",
	"ieqcharts/chartheader.html",
	"ieqcharts/chart.html",
	"ieqcharts/line.html",
	"ieqcharts/hearts.html",
	"ieqcharts/devicestatus.html",
	"ieqcharts/chart_a.html",
	"ieqcharts/scriptdonut_a.html",
	"ieqcharts/scripthbarstacked_a.html",
	"ieqcharts/line_a.html",
	"ieqcharts/scriptline_a.html"))

// displayLocation is the timezone charts render times in.
// Loaded once at startup; override with the IEQ_TZ environment variable.
var displayLocation *time.Location

//=============================================================================
// HTTP Handlers

// defaultHandler handles /ping route, renders a simple hello
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Host: %s Path: %s\n", r.Host, r.URL.Path)
	fmt.Fprintln(w, "guten tag! ich bin am Leben")
}

// getDeviceIDFromURL extracts and returns the Device ID from the URL query.
// If not found, error and empty string are returned.
func getDeviceIDFromURL(r *http.Request) (string, error) {
	result := r.URL.Query().Get("device_id")
	if result == "" {
		return "", errors.New("device_id param missing in URL query string")
	}
	return result, nil
}

// serveError logs the detail server-side and sends a generic message with the
// proper status code, so internal errors are not leaked to clients.
func serveError(w http.ResponseWriter, err error, status int) {
	log.Println(err)
	if errors.Is(err, db.ErrNoRecord) {
		http.Error(w, "no data found for device", http.StatusNotFound)
		return
	}
	http.Error(w, http.StatusText(status), status)
}

// ieqNumbersHandler renders latest IEQ scores and metrics numbers
func ieqNumbersHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Host: %s Path: %s\n", r.Host, r.URL.Path)
	ctx := r.Context()

	deviceID, err := getDeviceIDFromURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbmetrics, err := db.ReadLatestMetrics(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	dbmetricscore, err := db.ReadLatestMetricScores(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	dbieqscore, err := db.ReadLatestIeqScores(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	devInfo, err := db.ReadLastDeviceStatus(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	if devInfo.Status == 0 {
		if err := templates.ExecuteTemplate(w, "status", devInfo); err != nil {
			log.Println(err)
		}
	}

	if err := templates.ExecuteTemplate(w, "ieqscores", dbieqscore); err != nil {
		log.Println(err)
	}
	if err := templates.ExecuteTemplate(w, "metricscores", dbmetricscore); err != nil {
		log.Println(err)
	}
	if err := templates.ExecuteTemplate(w, "metrics", dbmetrics); err != nil {
		log.Println(err)
	}
}

// ieqchartsHandler renders latest IEQ donut and side-by-side chart
func ieqchartsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Host: %s Path: %s\n", r.Host, r.URL.Path)
	ctx := r.Context()

	deviceID, err := getDeviceIDFromURL(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dbmetricsList, err := db.ReadMetrics(ctx, deviceID, 10)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	dbmetricscore, err := db.ReadLatestMetricScores(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	dbieqscore, err := db.ReadLatestIeqScores(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	devInfo, err := db.ReadLastDeviceStatus(ctx, deviceID)
	if err != nil {
		serveError(w, err, http.StatusInternalServerError)
		return
	}

	if devInfo.Status == 0 {
		if err := templates.ExecuteTemplate(w, "devicestatus", devInfo); err != nil {
			log.Println(err)
		}
	}

	// using anonymous struct to pass data into go template
	scores := struct {
		IeqScores    mdl.IeqScore
		MetricScores mdl.MetricScore
	}{dbieqscore, dbmetricscore}

	// show IEQ elements depending on weightings, otherwise just IAQ elements
	if dbieqscore.LightingWeighting > 0 && dbieqscore.NoiseWeighting > 0 {
		if err := templates.ExecuteTemplate(w, "chart", scores); err != nil {
			log.Println(err)
		}
	} else {
		if err := templates.ExecuteTemplate(w, "chart_a", scores); err != nil {
			log.Println(err)
		}
	}

	if err := templates.ExecuteTemplate(w, "hearts", scores); err != nil {
		log.Println(err)
	}

	// dbmetricsList is in time descending order; the line chart needs time
	// ascending order
	slices.Reverse(dbmetricsList)

	// just a local struct type to be passed to gotemplates
	metrics := struct {
		Times        []string
		Temperatures []float64
		Humidities   []float64
		CO2s         []float64
		VOCs         []float64
		PM25s        []float64
		Visuals      []float64
		Acoustics    []float64
	}{}

	for _, m := range dbmetricsList {
		local := m.CreatedOn.In(displayLocation)
		metrics.Times = append(metrics.Times, local.Format("15:04"))
		metrics.Temperatures = append(metrics.Temperatures, m.Temperature)
		metrics.Humidities = append(metrics.Humidities, m.Humidity)
		metrics.CO2s = append(metrics.CO2s, m.CO2)
		metrics.VOCs = append(metrics.VOCs, m.VOC)
		metrics.PM25s = append(metrics.PM25s, m.PM25)
		metrics.Visuals = append(metrics.Visuals, m.Lighting)
		metrics.Acoustics = append(metrics.Acoustics, m.Noise)
	}

	// show IEQ elements depending on weightings, otherwise just IAQ elements
	if dbieqscore.LightingWeighting > 0 && dbieqscore.NoiseWeighting > 0 {
		if err := templates.ExecuteTemplate(w, "line", metrics); err != nil {
			log.Println(err)
		}
	} else {
		if err := templates.ExecuteTemplate(w, "line_a", metrics); err != nil {
			log.Println(err)
		}
	}
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Println("arguments expected: listening Port")
		return
	}
	port := ":" + args[1]

	tz := os.Getenv("IEQ_TZ")
	if tz == "" {
		tz = "Asia/Singapore"
	}
	var err error
	displayLocation, err = time.LoadLocation(tz)
	if err != nil {
		log.Fatalf("time.LoadLocation %s: %v", tz, err)
	}

	mux := http.NewServeMux()

	// http://127.0.0.1:<port>/ping
	mux.HandleFunc("GET /ping", defaultHandler)

	// Example to get the IEQ numbers
	// http://localhost:<port>/ieq/numbers?device_id=awair-omni_18453
	mux.HandleFunc("GET /ieq/numbers", ieqNumbersHandler)

	// Example to get the IEQ charts
	// http://localhost:<port>/ieq/device?device_id=awair-omni_18453
	mux.HandleFunc("GET /ieq/device", ieqchartsHandler)

	// Temporary, this is for static mockup test only
	mux.Handle("GET /moqup/", http.StripPrefix("/moqup/", http.FileServer(http.Dir("./static"))))

	server := &http.Server{Addr: port, Handler: mux}

	// shut down gracefully on Ctrl-C / SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("IEQ up and running at port%s ...\n", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
