package vector

import (
	"math"
	"runtime"
	"testing"
)

func TestSIMDAvailability(t *testing.T) {
	// This just checks that our SIMD detection code works
	if !IsSIMDAvailable() {
		t.Logf("SIMD support not detected - SIMD tests will be skipped")
		// Note: This is not an error or test failure. SIMD tests are designed
		// to be skipped on platforms without SIMD support. The fallback to regular
		// implementations ensures the code still works correctly.
	} else {
		t.Logf("SIMD support detected for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// TestSIMDVsRegular compares SIMD and regular implementations to ensure they're equivalent
func TestSIMDVsRegular(t *testing.T) {
	if !IsSIMDAvailable() {
		t.Skip("Skipping SIMD tests as SIMD is not available")
	}
	
	// Test vectors of various dimensions
	dims := []int{4, 16, 128, 512}
	
	for _, dim := range dims {
		t.Run("dim_"+string(rune(dim)), func(t *testing.T) {
			// Create vectors
			v1 := make([]float32, dim)
			v2 := make([]float32, dim)
			
			// Fill with test data
			for i := 0; i < dim; i++ {
				v1[i] = float32(i) / float32(dim)
				v2[i] = float32(dim-i) / float32(dim)
			}
			
			// Compare cosine distance implementations
			regularCosine := CosineDistance(v1, v2)
			simdCosine := SIMDCosineDistance(v1, v2)
			
			epsilon := float32(0.0001)
			if math.Abs(float64(regularCosine-simdCosine)) > float64(epsilon) {
				t.Errorf("Cosine distance mismatch: regular=%f, SIMD=%f", 
					regularCosine, simdCosine)
			}
			
			// Compare dot product implementations
			regularDot := DotProduct(v1, v2)
			simdDot := SIMDDotProduct(v1, v2)
			
			if math.Abs(float64(regularDot-simdDot)) > float64(epsilon) {
				t.Errorf("Dot product mismatch: regular=%f, SIMD=%f", 
					regularDot, simdDot)
			}
			
			// Compare Euclidean distance
			regularEuclid := EuclideanDistance(v1, v2)
			simdEuclid := SIMDEuclideanDistance(v1, v2)
			
			if math.Abs(float64(regularEuclid-simdEuclid)) > float64(epsilon) {
				t.Errorf("Euclidean distance mismatch: regular=%f, SIMD=%f", 
					regularEuclid, simdEuclid)
			}
		})
	}
}

func TestSIMDBatchOperations(t *testing.T) {
	if !IsSIMDAvailable() {
		t.Skip("Skipping SIMD tests as SIMD is not available")
	}
	
	// Test query and vectors
	dim := 128
	query := make([]float32, dim)
	numVectors := 10
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
	
	// Compare batch cosine results
	regularBatch := BatchCosineDistance(query, vectors)
	simdBatch := SIMDBatchCosineDistance(query, vectors)
	
	if len(regularBatch) != len(simdBatch) {
		t.Fatalf("Batch result length mismatch: regular=%d, SIMD=%d", 
			len(regularBatch), len(simdBatch))
	}
	
	for i := range regularBatch {
		epsilon := float32(0.0001)
		if math.Abs(float64(regularBatch[i]-simdBatch[i])) > float64(epsilon) {
			t.Errorf("Batch cosine result[%d] mismatch: regular=%f, SIMD=%f", 
				i, regularBatch[i], simdBatch[i])
		}
	}
}

// Benchmark SIMD vs regular implementations
func BenchmarkSIMDCosineDistance(b *testing.B) {
	if !IsSIMDAvailable() {
		b.Skip("Skipping SIMD benchmark as SIMD is not available")
	}
	
	dim := 128
	v1 := make([]float32, dim)
	v2 := make([]float32, dim)
	
	// Initialize vectors
	for i := 0; i < dim; i++ {
		v1[i] = float32(i) / float32(dim)
		v2[i] = float32(dim-i) / float32(dim)
	}
	
	b.Run("Regular", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			CosineDistance(v1, v2)
		}
	})
	
	b.Run("SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			SIMDCosineDistance(v1, v2)
		}
	})
}

func BenchmarkSIMDDotProduct(b *testing.B) {
	if !IsSIMDAvailable() {
		b.Skip("Skipping SIMD benchmark as SIMD is not available")
	}
	
	dim := 128
	v1 := make([]float32, dim)
	v2 := make([]float32, dim)
	
	// Initialize vectors
	for i := 0; i < dim; i++ {
		v1[i] = float32(i) / float32(dim)
		v2[i] = float32(dim-i) / float32(dim)
	}
	
	b.Run("Regular", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			DotProduct(v1, v2)
		}
	})
	
	b.Run("SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			SIMDDotProduct(v1, v2)
		}
	})
}

func BenchmarkSIMDBatchCosine(b *testing.B) {
	if !IsSIMDAvailable() {
		b.Skip("Skipping SIMD benchmark as SIMD is not available")
	}
	
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
	
	b.Run("Regular", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			BatchCosineDistance(query, vectors)
		}
	})
	
	b.Run("SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			SIMDBatchCosineDistance(query, vectors)
		}
	})
}