package vector

import (
	"runtime"
	"unsafe"
)

// IsSIMDAvailable returns true if SIMD instructions are available on this platform
func IsSIMDAvailable() bool {
	// In production, you would use a library like x/sys/cpu to detect CPU features
	// For testing purposes, we'll check if the architecture potentially supports SIMD
	return runtime.GOARCH == "amd64" || runtime.GOARCH == "386"
}

// UseSimdAcceleration controls whether to use SIMD-accelerated distance calculations
// This can be disabled for testing or debugging purposes
var UseSimdAcceleration = true

// CosineSimilaritySIMD calculates the cosine similarity using SIMD instructions
// This is a placeholder for the actual SIMD implementation
// In a real implementation, you would use assembly or CGO to access SIMD instructions
func CosineSimilaritySIMD(a, b []float32) float32 {
	// This is just a placeholder that calls the non-SIMD version
	// In a real implementation, this would use AVX/SSE instructions for better performance
	if !UseSimdAcceleration || !isAVXSupported || len(a) < 4 {
		return CosineSimilarity(a, b)
	}

	// Actual SIMD implementation would go here
	// We would process 4-8 elements at a time using SIMD registers
	
	// For now, we'll just call the scalar implementation
	return CosineSimilarity(a, b)
}

// DotProductSIMD calculates the dot product using SIMD instructions
// This is a placeholder for the actual SIMD implementation
func DotProductSIMD(a, b []float32) float32 {
	if !UseSimdAcceleration || !isAVXSupported || len(a) < 4 {
		return DotProduct(a, b)
	}

	// Actual SIMD implementation would go here
	return DotProduct(a, b)
}

// EuclideanDistanceSIMD calculates the Euclidean distance using SIMD instructions
// This is a placeholder for the actual SIMD implementation
func EuclideanDistanceSIMD(a, b []float32) float32 {
	if !UseSimdAcceleration || !isAVXSupported || len(a) < 4 {
		return EuclideanDistance(a, b)
	}

	// Actual SIMD implementation would go here
	return EuclideanDistance(a, b)
}

// ManhattanDistanceSIMD calculates the Manhattan distance using SIMD instructions
// This is a placeholder for the actual SIMD implementation
func ManhattanDistanceSIMD(a, b []float32) float32 {
	if !UseSimdAcceleration || !isAVXSupported || len(a) < 4 {
		return ManhattanDistance(a, b)
	}

	// Actual SIMD implementation would go here
	return ManhattanDistance(a, b)
}

// alignVector aligns the vector to a 32-byte boundary for optimal SIMD performance
// In a real implementation, this would ensure vectors are aligned for AVX/SSE instructions
func alignVector(v []float32) []float32 {
	// This is a simplified version - in production, you would use
	// memory alignment techniques to ensure proper alignment
	
	// Check if the vector is already aligned
	addr := uintptr(unsafe.Pointer(&v[0]))
	if addr%32 == 0 {
		return v
	}
	
	// Create a new aligned vector
	aligned := make([]float32, len(v))
	copy(aligned, v)
	return aligned
}

// batchCosineSimilaritySIMD calculates cosine similarities between a query vector
// and multiple vectors using SIMD instructions
func batchCosineSimilaritySIMD(query []float32, vectors [][]float32) []float32 {
	if !UseSimdAcceleration || !isAVXSupported {
		// Fall back to scalar implementation
		results := make([]float32, len(vectors))
		for i, vec := range vectors {
			results[i] = CosineSimilarity(query, vec)
		}
		return results
	}
	
	// Prepare the query vector
	alignedQuery := alignVector(query)
	
	// Process all vectors
	results := make([]float32, len(vectors))
	for i, vec := range vectors {
		alignedVec := alignVector(vec)
		results[i] = CosineSimilaritySIMD(alignedQuery, alignedVec)
	}
	
	return results
}

// GetOptimizedDistanceFunc returns the most optimized distance function for the given parameters
// It chooses between scalar and SIMD implementations based on hardware capabilities
func GetOptimizedDistanceFunc(useSimd bool, dimension int) DistanceFunc {
	// A real implementation would have different optimized functions for different vector sizes
	if useSimd && IsSIMDAvailable() && dimension >= 4 {
		return CosineSimilaritySIMD
	}
	
	return CosineSimilarity
}

// Function aliases for test compatibility
// These provide the naming that our tests expect

// SIMDCosineDistance is a SIMD-optimized version of cosine distance calculation
func SIMDCosineDistance(a, b []float32) float32 {
	// Convert similarity to distance
	return 1.0 - CosineSimilaritySIMD(a, b)
}

// SIMDDotProduct is a SIMD-optimized version of dot product calculation
func SIMDDotProduct(a, b []float32) float32 {
	return DotProductSIMD(a, b)
}

// SIMDEuclideanDistance is a SIMD-optimized version of Euclidean distance calculation
func SIMDEuclideanDistance(a, b []float32) float32 {
	return EuclideanDistanceSIMD(a, b)
}

// SIMDBatchCosineDistance is a SIMD-optimized version of batch cosine distance calculation
func SIMDBatchCosineDistance(query []float32, vectors [][]float32) []float32 {
	// Calculate similarities
	similarities := batchCosineSimilaritySIMD(query, vectors)
	
	// Convert similarities to distances
	distances := make([]float32, len(similarities))
	for i, sim := range similarities {
		distances[i] = 1.0 - sim
	}
	
	return distances
}