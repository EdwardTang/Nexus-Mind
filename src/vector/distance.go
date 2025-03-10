package vector

import (
	"errors"
	"math"

	"course/models"
)

// DistanceFunc is a function type that calculates distance between two vectors
type DistanceFunc func(a, b []float32) float32

// GetDistanceFunc returns the appropriate distance function for the given metric
func GetDistanceFunc(metric models.DistanceMetric) (DistanceFunc, error) {
	switch metric {
	case models.Cosine:
		return CosineSimilarity, nil
	case models.DotProduct:
		return DotProduct, nil
	case models.Euclidean:
		return EuclideanDistance, nil
	case models.Manhattan:
		return ManhattanDistance, nil
	default:
		return nil, errors.New("unsupported distance metric")
	}
}

// CosineSimilarity calculates the cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return -1 // Error case, different dimensions
	}
	
	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0 // Handle zero vectors
	}
	
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// DotProduct calculates the dot product between two vectors
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0 // Error case, different dimensions
	}
	
	var dotProduct float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	
	return dotProduct
}

// EuclideanDistance calculates the Euclidean distance between two vectors
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1)) // Error case, different dimensions
	}
	
	var sumSquares float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sumSquares += diff * diff
	}
	
	return float32(math.Sqrt(float64(sumSquares)))
}

// ManhattanDistance calculates the Manhattan (L1) distance between two vectors
func ManhattanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1)) // Error case, different dimensions
	}
	
	var sumAbsDiff float32
	for i := 0; i < len(a); i++ {
		sumAbsDiff += float32(math.Abs(float64(a[i] - b[i])))
	}
	
	return sumAbsDiff
}

// NormalizeVector normalizes a vector in-place to have unit length (L2 norm)
func NormalizeVector(v []float32) {
	var sumSquares float32
	for _, val := range v {
		sumSquares += val * val
	}
	
	if sumSquares == 0 {
		return // Can't normalize a zero vector
	}
	
	norm := float32(math.Sqrt(float64(sumSquares)))
	for i := range v {
		v[i] /= norm
	}
}

// PrecomputeNorms calculates and stores L2 norms for a batch of vectors
// This is useful for optimizing distance calculations
func PrecomputeNorms(vectors [][]float32) []float32 {
	norms := make([]float32, len(vectors))
	
	for i, vec := range vectors {
		var sumSquares float32
		for _, val := range vec {
			sumSquares += val * val
		}
		norms[i] = float32(math.Sqrt(float64(sumSquares)))
	}
	
	return norms
}

// CosineSimilarityWithNorms calculates cosine similarity using precomputed norms
func CosineSimilarityWithNorms(a, b []float32, normA, normB float32) float32 {
	if len(a) != len(b) {
		return -1 // Error case, different dimensions
	}
	
	var dotProduct float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0 // Handle zero vectors
	}
	
	return dotProduct / (normA * normB)
}

// CosineSimilarityNormalized calculates the cosine similarity between 
// two normalized vectors (optimized, assumes unit vectors)
func CosineSimilarityNormalized(a, b []float32) float32 {
	if len(a) != len(b) {
		return -1 // Error case, different dimensions
	}
	
	var dotProduct float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	
	return dotProduct
}

// BatchDistance calculates distances between one query vector and multiple vectors
func BatchDistance(query []float32, vectors [][]float32, metric models.DistanceMetric) ([]float32, error) {
	distFunc, err := GetDistanceFunc(metric)
	if err != nil {
		return nil, err
	}
	
	results := make([]float32, len(vectors))
	for i, vec := range vectors {
		results[i] = distFunc(query, vec)
	}
	
	return results, nil
}

// IsHigherBetter returns true if a higher value is better for the given metric
// Used for scoring and sorting search results
func IsHigherBetter(metric models.DistanceMetric) bool {
	switch metric {
	case models.Cosine, models.DotProduct:
		return true // Higher is better for similarity metrics
	case models.Euclidean, models.Manhattan:
		return false // Lower is better for distance metrics
	default:
		return true // Default assumption
	}
}

// NormalizeScore converts a raw distance/similarity value to a normalized score (0-1)
// where 1 is the best match and 0 is the worst
func NormalizeScore(rawValue float32, metric models.DistanceMetric) float32 {
	switch metric {
	case models.Cosine:
		// Convert from [-1,1] to [0,1]
		return (rawValue + 1) / 2
	case models.DotProduct:
		// DotProduct can be unbounded, so this is a simplification
		// In a real implementation, you'd normalize based on vector magnitudes
		if rawValue <= 0 {
			return 0
		}
		if rawValue >= 1 {
			return 1
		}
		return rawValue
	case models.Euclidean:
		// Euclidean distance is unbounded above, so we need some heuristic
		// One approach is to use an exponential decay function
		return float32(math.Exp(-float64(rawValue)))
	case models.Manhattan:
		// Similar to Euclidean
		return float32(math.Exp(-float64(rawValue) * 0.5))
	default:
		return 0.5 // Default value if unknown metric
	}
}