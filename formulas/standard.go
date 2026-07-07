package formulas

import (
	"fmt"
)

// StandardFormula defines a type to contain the standard formula and implement Score interface
type StandardFormula struct {
	name    string
	formula *Formula
}

// Setup implements interface Scorer.Setup()
// Setup will populate the scores ranges used by Score().
func (f *StandardFormula) Setup(n string, minVal float64, maxVal float64) {
	f.name = n
	f.formula = newFormula(minVal, maxVal)
}

// Score implements interface Scorer.Score()
func (f *StandardFormula) Score(v float64) (result float64, ok bool) {

	// in target range
	if v >= f.formula.min && v <= f.formula.max {
		return 100, true
	}

	// higher than best average
	if v > f.formula.avg {
		rr := (v - f.formula.avg) / f.formula.avg * 100
		result, ok = f.formula.ranges.Search(rr)
	} else if v < f.formula.avg {
		rr := (f.formula.avg - v) / f.formula.avg * 100
		result, ok = f.formula.ranges.Search(rr)
	} else {
		return 0.0, false
	}
	return
}

// Name implements interface Scorer.Name()
func (f *StandardFormula) Name() string {
	return f.name
}

// ToString implements interface Stringer.ToString()
func (f *StandardFormula) ToString() string {
	return fmt.Sprintf("Formula:%s %s", f.name, stringify(f.formula))
}
