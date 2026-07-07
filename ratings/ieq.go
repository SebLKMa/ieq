package ratings

// IEQRating contains a Rating to represent a overall IEQ rating
type IEQRating struct {
	rating Rating
}

// Setup implements interface Rateable.Setup.
// This function sets up the IEQ component weighting for its rating done by SetRating().
func (r *IEQRating) Setup(n string, w float64) {
	r.rating = Rating{}
	r.rating.name = n
	r.rating.weighting = w
}

// AddIndex implements interface Rateable.AddIndex
// Adds an IEQ index and its score value for subsequent rating in SetRating().
func (r *IEQRating) AddIndex(n string, v float64) error {
	return r.rating.AddIndex(n, v)
}

// Indices returns a list of added indices
func (r *IEQRating) Indices() []Index {
	return r.rating.indices
}

// SetRating implements interface Rateable.SetRating
// Computes the overall IEQ Rating score.
func (r *IEQRating) SetRating() {
	if len(r.rating.indices) == 0 {
		return
	}

	sum := float64(0.0)
	for _, i := range r.rating.indices {
		sum += i.score
	}
	if sum == 0.0 {
		return
	}

	// score = thermal + iaq + light + noise
	r.rating.score = sum
}

// Name implements Rateable.Name
// Returns the Rating's Name.
func (r *IEQRating) Name() string {
	return r.rating.name
}

// Weighting implements Rateable.Weighting
// Returns the Rating's Weighting.
func (r *IEQRating) Weighting() float64 {
	return r.rating.weighting
}

// Rate implements Rateable.Rate
// Returns the computed rating score.
func (r *IEQRating) Rate() float64 {
	return r.rating.score
}
