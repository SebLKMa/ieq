package formulas

import (
	"fmt"
)

// MinIsGoodFormula defines a type to contain the minimum is good formula and implement Score interface
type MinIsGoodFormula struct {
	mingood StandardFormula
}

// Setup implements interface Scorer.Setup().
// Setup will populate the scores ranges used by Score().
func (f *MinIsGoodFormula) Setup(n string, minVal float64, maxVal float64) {
	f.mingood.Setup(n, minVal, maxVal)
}

// Score implements interface Scorer.Score()
func (f *MinIsGoodFormula) Score(v float64) (result float64, ok bool) {

	// for minimum is good formula, anything less than target max is good
	if v <= f.mingood.formula.max {
		return 100, true
	}

	// higher than best average
	if v > f.mingood.formula.avg {
		rr := (v - f.mingood.formula.avg) / f.mingood.formula.avg * 100
		result, ok = f.mingood.formula.ranges.Search(rr)
		if !ok {
			return 0.0, true // zero score for out of generated ranges
		}
	} else {
		return 0.0, false // should not reach here
	}
	return
}

// Name implements interface Scorer.Name()
func (f *MinIsGoodFormula) Name() string {
	return f.mingood.name
}

// ToString implements interface Stringer.ToString()
func (f *MinIsGoodFormula) ToString() string {
	return fmt.Sprintf("Formula:%s %s", f.mingood.name, stringify(f.mingood.formula))
}
