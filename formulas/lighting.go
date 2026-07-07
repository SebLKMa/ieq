package formulas

import (
	"fmt"
)

// LightingFormula defines a type to contain the lighting formula and implement Score interface.
// Use NewLightingFormula to construct it with the required scale.
type LightingFormula struct {
	name    string
	formula *Formula
	scale   float64
}

// NewLightingFormula returns a LightingFormula with the given scale.
// A scale <= 0 falls back to the default of 1.
func NewLightingFormula(scale float64) *LightingFormula {
	f := &LightingFormula{}
	f.SetScale(scale)
	return f
}

// SetScale sets the scale for Setup to generate ranges.
// SetScale has to be called before Setup.
// If not called, the default scale is set to 1.
//
// Deprecated: use NewLightingFormula, which removes the call-order requirement.
func (f *LightingFormula) SetScale(sc float64) {
	f.scale = sc
}

// Setup implements interface Scorer.Setup()
// Setup will populate the scores ranges used by Score().
// If scale is not set, the default scale is set to 1.
func (f *LightingFormula) Setup(n string, minVal float64, maxVal float64) {
	if f.scale <= 0 {
		f.scale = 1
	}
	f.name = n
	f.formula = &Formula{min: minVal, max: maxVal}
	f.formula.avg = (f.formula.min + f.formula.max) / 2
	f.formula.rrange = (f.formula.max - f.formula.avg) / f.formula.avg * 100

	// generate lux ranges outward from the average: each step scales the
	// bounds up (incremental) and down (decremental) and lowers the score
	incrementalfrom := f.formula.avg
	incrementalto := incrementalfrom * f.scale
	decrementalfrom := f.formula.avg
	decrementalto := decrementalfrom / f.scale
	chunks := 10
	diff := 12.5 // interval
	score := 100.0
	for i := 1; i < chunks; i++ {
		f.formula.ranges.Insert(incrementalfrom, incrementalto, score) // from - to
		f.formula.ranges.Insert(decrementalto, decrementalfrom, score) // to - from
		score = score - diff
		incrementalfrom = incrementalto
		incrementalto *= f.scale
		decrementalfrom = decrementalto
		decrementalto /= f.scale
	}
}

// Score implements interface Scorer.Score()
func (f *LightingFormula) Score(v float64) (result float64, ok bool) {

	result, ok = f.formula.ranges.Search(v)
	if !ok {
		// score is 0 when lux value is out of ranges
		return result, true
	}

	return
}

// Name implements interface Scorer.Name()
func (f *LightingFormula) Name() string {
	return f.name
}

// ToString implements interface Stringer.ToString()
func (f *LightingFormula) ToString() string {
	return fmt.Sprintf("Formula:%s %s Scale:%g", f.name, stringify(f.formula), f.scale)
}
