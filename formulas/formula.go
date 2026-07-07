package formulas

import (
	"fmt"

	st "github.com/seblkma/ieq/utils/skiptree"
)

// Formula defines the arguments to setup a Scorer object
type Formula struct {
	min    float64
	max    float64
	avg    float64
	rrange float64
	ranges st.ItemSkipTree
}

// newFormula computes the average and relative range for the recommended
// [minVal, maxVal] band and generates the relative score ranges used by Score.
// Scores run from 100 down in equal steps across contiguous relative ranges.
func newFormula(minVal float64, maxVal float64) *Formula {
	f := &Formula{min: minVal, max: maxVal}
	f.avg = (f.min + f.max) / 2
	f.rrange = (f.max - f.avg) / f.avg * 100

	from := 0.0
	to := f.rrange
	high := 100
	low := -10
	chunks := 11
	diff := (high - low) / chunks
	score := high
	for i := 1; i < chunks; i++ {
		f.ranges.Insert(from, to, float64(score))
		score = score - diff
		from = to
		to += f.rrange
	}
	f.ranges.Insert(from, to, float64(score))
	return f
}

// Returns the contents of a Formula
// Using pointer to avoid copying in case of large skiptree
func stringify(f *Formula) string {
	return fmt.Sprintf("Min=%g Max=%g Avg=%g Rrange=%g", f.min, f.max, f.avg, f.rrange)
}
