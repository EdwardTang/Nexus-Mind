package vector

import (
	"course/models"
	"math"
	"testing"
)

func TestCosineDistance(t *testing.T) {
	// Test cases with expected distances
	testCases := []struct {
		a        []float32
		b        []float32
		expected float32
	}{
		{[]float32{1, 0, 0}, []float32{1, 0, 0}, 0.0}, // Same vector
		{[]float32{1, 0, 0}, []float32{0, 1, 0}, 1.0}, // Orthogonal
		{[]float32{1, 0, 0}, []float32{-1, 0, 0}, 2.0}, // Opposite
		{[]float32{1, 1, 0}, []float32{1, 0, 0}, 0.29289323}, // 45 degrees
		{[]float32{0, 0, 0}, []float32{1, 1, 1}, 1.0}, // Zero vector
	}

	for i, tc := range testCases {
		result := CosineDistance(tc.a, tc.b)
		
		// Use an epsilon for floating point comparison
		epsilon := float32(0.0001)
		if math.Abs(float64(result-tc.expected)) > float64(epsilon) {
			t.Errorf("Case %d: CosineDistance(%v, %v) = %f, expected %f", 
				i, tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestEuclideanDistance(t *testing.T) {
	// Test cases with expected distances
	testCases := []struct {
		a        []float32
		b        []float32
		expected float32
	}{
		{[]float32{1, 0, 0}, []float32{1, 0, 0}, 0.0}, // Same vector
		{[]float32{0, 0, 0}, []float32{1, 0, 0}, 1.0}, // Unit distance
		{[]float32{1, 1, 0}, []float32{4, 5, 0}, 5.0}, // Pythagorean triple (3,4,5)
		{[]float32{-1, -1, -1}, []float32{1, 1, 1}, 3.4641016}, // 2*sqrt(3)
	}

	for i, tc := range testCases {
		result := EuclideanDistance(tc.a, tc.b)
		
		// Use an epsilon for floating point comparison
		epsilon := float32(0.0001)
		if math.Abs(float64(result-tc.expected)) > float64(epsilon) {
			t.Errorf("Case %d: EuclideanDistance(%v, %v) = %f, expected %f", 
				i, tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestManhattanDistance(t *testing.T) {
	// Test cases with expected distances
	testCases := []struct {
		a        []float32
		b        []float32
		expected float32
	}{
		{[]float32{1, 0, 0}, []float32{1, 0, 0}, 0.0}, // Same vector
		{[]float32{0, 0, 0}, []float32{1, 0, 0}, 1.0}, // Unit distance
		{[]float32{1, 1, 1}, []float32{2, 3, 4}, 6.0}, // Sum of coordinate differences
		{[]float32{-1, -1, -1}, []float32{1, 1, 1}, 6.0}, // Absolute differences
	}

	for i, tc := range testCases {
		result := ManhattanDistance(tc.a, tc.b)
		
		// Use an epsilon for floating point comparison
		epsilon := float32(0.0001)
		if math.Abs(float64(result-tc.expected)) > float64(epsilon) {
			t.Errorf("Case %d: ManhattanDistance(%v, %v) = %f, expected %f", 
				i, tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestDotProduct(t *testing.T) {
	// Test cases with expected dot products
	testCases := []struct {
		a        []float32
		b        []float32
		expected float32
	}{
		{[]float32{1, 0, 0}, []float32{1, 0, 0}, 1.0}, // Unit vectors aligned
		{[]float32{1, 0, 0}, []float32{0, 1, 0}, 0.0}, // Orthogonal
		{[]float32{1, 2, 3}, []float32{4, 5, 6}, 32.0}, // 1*4 + 2*5 + 3*6 = 32
		{[]float32{-1, -1, -1}, []float32{1, 1, 1}, -3.0}, // Opposite directions
	}

	for i, tc := range testCases {
		result := DotProduct(tc.a, tc.b)
		
		// Use an epsilon for floating point comparison
		epsilon := float32(0.0001)
		if math.Abs(float64(result-tc.expected)) > float64(epsilon) {
			t.Errorf("Case %d: DotProduct(%v, %v) = %f, expected %f", 
				i, tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestDistanceFunc(t *testing.T) {
	// Test getting the correct distance function for each metric
	testVectors := []float32{1, 2, 3}
	
	// Test each metric
	metrics := []struct {
		metric   models.DistanceMetric
		expected float32
	}{
		{models.Cosine, CosineDistance(testVectors, testVectors)},
		{models.Euclidean, EuclideanDistance(testVectors, testVectors)},
		{models.Manhattan, ManhattanDistance(testVectors, testVectors)},
		{models.DotProduct, 1.0 - DotProduct(testVectors, testVectors)}, // Note the 1.0 - DP for distance
	}
	
	for _, tc := range metrics {
		distFunc := GetDistanceFunc(tc.metric)
		result := distFunc(testVectors, testVectors)
		
		// For identity comparison, all should be 0 or 1-dot product
		if tc.metric == models.DotProduct {
			// The distance for dot product is 1.0 - dot product for same vector
			expected := 1.0 - DotProduct(testVectors, testVectors)
			if result != expected {
				t.Errorf("GetDistanceFunc(%s) returned function that gave %f for identical vectors, expected %f", 
					tc.metric.String(), result, expected)
			}
		} else {
			// For all other metrics, distance to self should be 0
			if result != 0.0 {
				t.Errorf("GetDistanceFunc(%s) returned function that gave %f for identical vectors, expected 0.0", 
					tc.metric.String(), result)
			}
		}
	}
}

func TestBatchDistanceCalculation(t *testing.T) {
	// Test vectors
	query := []float32{1, 0, 0}
	vectors := [][]float32{
		{1, 0, 0},   // Same as query
		{0, 1, 0},   // Orthogonal
		{0.5, 0.5, 0}, // 45 degrees
	}
	
	// Expected cosine distances
	expectedCosine := []float32{0.0, 1.0, 0.29289323}
	
	// Calculate batch cosine distances
	cosineResults := BatchCosineDistance(query, vectors)
	
	// Check results
	if len(cosineResults) != len(vectors) {
		t.Errorf("BatchCosineDistance returned %d results, expected %d", 
			len(cosineResults), len(vectors))
	}
	
	for i, dist := range cosineResults {
		epsilon := float32(0.0001)
		if math.Abs(float64(dist-expectedCosine[i])) > float64(epsilon) {
			t.Errorf("BatchCosineDistance result[%d] = %f, expected %f", 
				i, dist, expectedCosine[i])
		}
	}
	
	// Test batch dot product
	expectedDot := []float32{1.0, 0.0, 0.5}
	dotResults := BatchDotProduct(query, vectors)
	
	for i, dot := range dotResults {
		epsilon := float32(0.0001)
		if math.Abs(float64(dot-expectedDot[i])) > float64(epsilon) {
			t.Errorf("BatchDotProduct result[%d] = %f, expected %f", 
				i, dot, expectedDot[i])
		}
	}
}

func TestNormalization(t *testing.T) {
	// Test vector
	vector := []float32{3, 4, 0} // Length should be 5
	
	// Normalize
	normalized := Normalize(vector)
	
	// Expected normalized values (3/5, 4/5, 0)
	expected := []float32{0.6, 0.8, 0.0}
	
	// Check normalization
	epsilon := float32(0.0001)
	for i, v := range normalized {
		if math.Abs(float64(v-expected[i])) > float64(epsilon) {
			t.Errorf("Normalize[%d] = %f, expected %f", i, v, expected[i])
		}
	}
	
	// Check length
	length := float32(0.0)
	for _, v := range normalized {
		length += v * v
	}
	length = float32(math.Sqrt(float64(length)))
	
	if math.Abs(float64(length-1.0)) > float64(epsilon) {
		t.Errorf("Normalized vector length = %f, expected 1.0", length)
	}
	
	// Test normalization of zero vector
	zeroVector := []float32{0, 0, 0}
	zeroNormalized := Normalize(zeroVector)
	
	// Zero vector should remain zero
	for i, v := range zeroNormalized {
		if v != 0 {
			t.Errorf("Normalized zero vector[%d] = %f, expected 0.0", i, v)
		}
	}
}

func TestVectorLength(t *testing.T) {
	// Test vectors
	testCases := []struct {
		vector   []float32
		expected float32
	}{
		{[]float32{1, 0, 0}, 1.0},
		{[]float32{3, 4, 0}, 5.0},
		{[]float32{0, 0, 0}, 0.0},
		{[]float32{1, 1, 1, 1}, 2.0}, // sqrt(4)
	}
	
	for i, tc := range testCases {
		length := VectorLength(tc.vector)
		
		epsilon := float32(0.0001)
		if math.Abs(float64(length-tc.expected)) > float64(epsilon) {
			t.Errorf("Case %d: VectorLength(%v) = %f, expected %f", 
				i, tc.vector, length, tc.expected)
		}
	}
}

// Benchmark distance function performance
func BenchmarkCosineDistance(b *testing.B) {
	dim := 128
	v1 := make([]float32, dim)
	v2 := make([]float32, dim)
	
	// Initialize vectors
	for i := 0; i < dim; i++ {
		v1[i] = float32(i) / float32(dim)
		v2[i] = float32(dim-i) / float32(dim)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineDistance(v1, v2)
	}
}

func BenchmarkEuclideanDistance(b *testing.B) {
	dim := 128
	v1 := make([]float32, dim)
	v2 := make([]float32, dim)
	
	// Initialize vectors
	for i := 0; i < dim; i++ {
		v1[i] = float32(i) / float32(dim)
		v2[i] = float32(dim-i) / float32(dim)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EuclideanDistance(v1, v2)
	}
}

func BenchmarkBatchCosineDistance(b *testing.B) {
	dim := 128
	numVectors := 100
	
	query := make([]float32, dim)
	vectors := make([][]float32, numVectors)
	
	// Initialize vectors
	for i := 0; i < dim; i++ {
		query[i] = float32(i) / float32(dim)
	}
	
	for i := 0; i < numVectors; i++ {
		vectors[i] = make([]float32, dim)
		for j := 0; j < dim; j++ {
			vectors[i][j] = float32(i*j) / float32(dim*numVectors)
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BatchCosineDistance(query, vectors)
	}
}