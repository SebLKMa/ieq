package ratings

import (
	"testing"
)

func TestStandardRating(t *testing.T) {
	// components weightings set to all 25
	thermalRating := Rating{}
	thermalRating.Setup("Thermal", 25)

	iaqRating := Rating{}
	iaqRating.Setup("IAQ", 25)

	lightingRating := Rating{}
	lightingRating.Setup("Lighting", 25)

	noiseRating := Rating{}
	noiseRating.Setup("Noise", 25)

	// all scores 100 percent
	mustAdd := func(r *Rating, n string, v float64) {
		t.Helper()
		if err := r.AddIndex(n, v); err != nil {
			t.Fatal(err)
		}
	}
	mustAdd(&thermalRating, "Temperature", 50)
	mustAdd(&thermalRating, "Humidity", 50)
	mustAdd(&iaqRating, "CO2", 30)
	mustAdd(&iaqRating, "VOC", 30)
	mustAdd(&iaqRating, "PM25", 40)
	mustAdd(&lightingRating, "Lighting", 100)
	mustAdd(&noiseRating, "Noise", 100)

	thermalRating.SetRating()
	iaqRating.SetRating()
	lightingRating.SetRating()
	noiseRating.SetRating()

	// expecting overall score also 100 percent
	ieqRating := IEQRating{}
	ieqRating.Setup("Overall IEQ", 1.0)
	ieqRating.AddIndex(thermalRating.Name(), thermalRating.Rate())
	ieqRating.AddIndex(iaqRating.Name(), iaqRating.Rate())
	ieqRating.AddIndex(lightingRating.Name(), lightingRating.Rate())
	ieqRating.AddIndex(noiseRating.Name(), noiseRating.Rate())
	ieqRating.SetRating()

	total := float64(0)
	for _, i := range ieqRating.Indices() {
		t.Logf("%v \n", i)
		total += i.score
	}
	if total != ieqRating.Rate() {
		t.Errorf("Unexpected Overall Rating %g\n", ieqRating.Rate())
	}
	t.Logf("%s Rating: %g\n", ieqRating.Name(), ieqRating.Rate())
}

// Regression test: each index score is a percentage in [0, 100], so several
// perfect scores in one component must all be accepted. The previous
// validation wrongly rejected an index when the scores summed past 100,
// silently dropping e.g. Humidity when Temperature already scored 100.
func TestAddIndexAcceptsMultiplePerfectScores(t *testing.T) {
	thermalRating := Rating{}
	thermalRating.Setup("Thermal", 25)

	if err := thermalRating.AddIndex("Temperature", 100); err != nil {
		t.Fatalf("unexpected error adding Temperature: %v", err)
	}
	if err := thermalRating.AddIndex("Humidity", 100); err != nil {
		t.Fatalf("unexpected error adding Humidity: %v", err)
	}

	thermalRating.SetRating()

	// average of (100, 100) weighted at 25 percent
	if got := thermalRating.Rate(); got != 25 {
		t.Errorf("Thermal rating = %g, want 25", got)
	}
}

func TestAddIndexRejectsOutOfRangeScore(t *testing.T) {
	r := Rating{}
	r.Setup("IAQ", 25)

	if err := r.AddIndex("CO2", 101); err == nil {
		t.Error("expected error for score above 100")
	}
	if err := r.AddIndex("CO2", -1); err == nil {
		t.Error("expected error for negative score")
	}
	if err := r.AddIndex("CO2", 100); err != nil {
		t.Errorf("unexpected error for boundary score 100: %v", err)
	}
}
