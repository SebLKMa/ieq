package ratings

import (
	"fmt"
)

// Index represents an index and its score
type Index struct {
	name  string
	score float64
}

// Rating represents rateable quality
type Rating struct {
	name      string
	weighting float64
	score     float64
	indices   []Index
}

// Setup implements interface Rateable.Setup
// Initializes the Rating with name and weighing.
func (r *Rating) Setup(n string, w float64) {
	r.name = n
	r.weighting = w
}

// AddIndex implements interface Rateable.AddIndex
// Adds an index and its score value.
// Each index score is a percentage and must be within [0, 100]; SetRating
// averages the index scores, so any number of indices may be added.
func (r *Rating) AddIndex(n string, v float64) error {
	if v < 0 || v > 100 {
		return fmt.Errorf("index %s score %g must be within 0 to 100 percent", n, v)
	}
	r.indices = append(r.indices, Index{name: n, score: v})
	return nil
}

// SetRating implements interface Rateable.SetRating
// Computes the Rating's score using its weighting.
func (r *Rating) SetRating() {
	count := float64(len(r.indices))
	if count == 0.0 {
		return
	}

	sum := float64(0.0)
	for _, i := range r.indices {
		sum += i.score
	}
	if sum == 0.0 {
		return
	}

	// e.g., score = ((r.Temperature + r.Humidity) / 2) * r.Rating.weighting / 100
	r.score = (sum / count) * r.weighting / 100
}

// Name implements Rateable.Name
// Returns the Rating's Name.
func (r *Rating) Name() string {
	return r.name
}

// Weighting implements Rateable.Weighting
// Returns the Rating's Weighting.
func (r *Rating) Weighting() float64 {
	return r.weighting
}

// Rate implements Rateable.Rate
// Returns the Rating's computed score.
func (r *Rating) Rate() float64 {
	return r.score
}
