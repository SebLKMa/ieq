package ratings

import (
	"fmt"

	intf "github.com/seblkma/ieq/interfaces"
)

// Setup sets up the concrete scorer
func Setup(sc intf.Scorer, name string, minVal float64, maxVal float64) {
	sc.Setup(name, minVal, maxVal)
}

// ComputeScore uses the provided concrete Scorer and measured value to compute its score.
// It returns an error when the Scorer cannot compute a score for the value, so
// callers can distinguish a failure from a legitimate zero score.
func ComputeScore(sc intf.Scorer, value float64) (float64, error) {
	score, ok := sc.Score(value)
	if !ok {
		return 0, fmt.Errorf("%s: unable to compute score for value %g", sc.Name(), value)
	}
	return score, nil
}

// PrintInfo is helper function to print description of a Stringer
func PrintInfo(str intf.Stringer) {
	fmt.Println(str.ToString())
}
