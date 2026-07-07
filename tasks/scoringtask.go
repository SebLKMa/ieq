package tasks

import (
	"log"
	"os"
	"strings"

	conf "github.com/seblkma/ieq/configs"
	fml "github.com/seblkma/ieq/formulas"
	rate "github.com/seblkma/ieq/ratings"
	util "github.com/seblkma/ieq/utils"
	"gopkg.in/yaml.v3"
)

// ScoringTask properties
type ScoringTask struct {
	TemperatureFormula *fml.StandardFormula
	HumidityFormula    *fml.StandardFormula
	Co2Formula         *fml.MinIsGoodFormula
	VocFormula         *fml.MinIsGoodFormula
	Pm25Formula        *fml.MinIsGoodFormula
	NoiseFormula       *fml.MinIsGoodFormula
	LightingFormula    *fml.LightingFormula
	Cfg                *conf.AppConfig
	Initialized        bool
}

// NewScoringTask constructs a new ScoreTask instance
// ScoreTask properties will be initialized from configuration (file or database).
// The vendor token can be supplied via the environment as <VENDOR>_TOKEN
// (e.g. AWAIR_TOKEN, UHOO_TOKEN) so no secret needs to live in the yaml file.
func NewScoringTask(configFile string) *ScoringTask {
	if !util.FileExists(configFile) {
		log.Fatalf("%s file not found in current directory.", configFile)
	}

	f, err := os.Open(configFile)
	if err != nil {
		log.Fatal(err)
	}
	decoder := yaml.NewDecoder(f)

	task := ScoringTask{Initialized: false}

	err = decoder.Decode(&task.Cfg)
	if err != nil {
		log.Fatal("Failed to decode config yaml file. ", err)
	}
	f.Close()

	if token := os.Getenv(strings.ToUpper(task.Cfg.VENDOR.Name) + "_TOKEN"); token != "" {
		task.Cfg.VENDOR.Token = token
	}
	// do not log the config itself: it carries the vendor token
	log.Printf("Loaded config %s for device %s (vendor %s, scheme %s)",
		configFile, task.Cfg.VENDOR.DeviceDisplayID, task.Cfg.VENDOR.Name, task.Cfg.WEIGHTINGS.Scheme)

	task.TemperatureFormula = &fml.StandardFormula{}
	task.HumidityFormula = &fml.StandardFormula{}
	task.Co2Formula = &fml.MinIsGoodFormula{}
	task.VocFormula = &fml.MinIsGoodFormula{}
	task.Pm25Formula = &fml.MinIsGoodFormula{}
	task.NoiseFormula = &fml.MinIsGoodFormula{}
	task.LightingFormula = fml.NewLightingFormula(task.Cfg.LIGHTING.Scale)

	rate.Setup(task.TemperatureFormula, "Temperature", task.Cfg.Temperature.Min, task.Cfg.Temperature.Max)
	rate.PrintInfo(task.TemperatureFormula)

	rate.Setup(task.HumidityFormula, "Humidity", task.Cfg.Humidity.Min, task.Cfg.Humidity.Max)
	rate.PrintInfo(task.HumidityFormula)

	rate.Setup(task.Co2Formula, "CO2", task.Cfg.CO2.Min, task.Cfg.CO2.Max)
	rate.PrintInfo(task.Co2Formula)

	rate.Setup(task.VocFormula, "VOC", task.Cfg.VOC.Min, task.Cfg.VOC.Max)
	rate.PrintInfo(task.VocFormula)

	rate.Setup(task.Pm25Formula, "PM25", task.Cfg.PM25.Min, task.Cfg.PM25.Max)
	rate.PrintInfo(task.Pm25Formula)

	rate.Setup(task.NoiseFormula, "Noise", task.Cfg.NOISE.Min, task.Cfg.NOISE.Max)
	rate.PrintInfo(task.NoiseFormula)

	rate.Setup(task.LightingFormula, "Lighting", task.Cfg.LIGHTING.Min, task.Cfg.LIGHTING.Max)
	rate.PrintInfo(task.LightingFormula)

	task.Initialized = true

	return &task
}
