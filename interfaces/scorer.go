package interfaces

// Scorer represents a object that can compute a score
type Scorer interface {
	Setup(n string, minVal float64, maxVal float64)
	Score(v float64) (result float64, ok bool)
	Name() string
}
