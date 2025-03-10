package index

import (
	"testing"

	"course/models"
	"course/vector"
)

func TestLinearIndex(t *testing.T) {
	// Create a test index
	dim := 4
	idx, err := NewLinearIndex(dim, models.Cosine)
	if err != nil {
		t.Fatalf("Failed to create linear index: %v", err)
	}

	// Test size of empty index
	if size := idx.Size(); size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}

	// Test dimension
	if d := idx.Dimension(); d != dim {
		t.Errorf("Expected dimension %d, got %d", dim, d)
	}

	// Create some test vectors
	vectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0, 0, 0}, map[string]interface{}{"category": "A"}),
		models.NewVector("v2", []float32{0, 1, 0, 0}, map[string]interface{}{"category": "B"}),
		models.NewVector("v3", []float32{0, 0, 1, 0}, map[string]interface{}{"category": "A"}),
		models.NewVector("v4", []float32{0, 0, 0, 1}, map[string]interface{}{"category": "B"}),
		models.NewVector("v5", []float32{0.5, 0.5, 0, 0}, map[string]interface{}{"category": "C"}),
	}

	// Insert vectors
	for _, v := range vectors {
		if err := idx.Insert(v); err != nil {
			t.Errorf("Error inserting vector %s: %v", v.ID, err)
		}
	}

	// Test size after insertion
	if size := idx.Size(); size != len(vectors) {
		t.Errorf("Expected size %d, got %d", len(vectors), size)
	}

	// Test search
	query := []float32{1, 0, 0, 0}
	k := 3
	results, err := idx.Search(query, k, nil, &models.SearchParams{})
	if err != nil {
		t.Errorf("Error searching: %v", err)
	}

	// Check number of results
	if len(results) != k {
		t.Errorf("Expected %d results, got %d", k, len(results))
	}

	// Check order of results (v1 should be the closest match to [1,0,0,0])
	if results[0].ID != "v1" {
		t.Errorf("Expected first result to be v1, got %s", results[0].ID)
	}

	// Test search with filter
	filter := models.NewAndFilter(
		models.NewEqualsCondition("category", "A"),
	)
	
	filteredResults, err := idx.Search(query, k, filter, &models.SearchParams{})
	if err != nil {
		t.Errorf("Error searching with filter: %v", err)
	}

	// Check filtered results
	if len(filteredResults) != 2 { // Only v1 and v3 have category A
		t.Errorf("Expected 2 filtered results, got %d", len(filteredResults))
	}

	for _, res := range filteredResults {
		if res.Vector.Metadata["category"] != "A" {
			t.Errorf("Expected category A, got %v", res.Vector.Metadata["category"])
		}
	}

	// Test deletion
	if err := idx.Delete("v1"); err != nil {
		t.Errorf("Error deleting vector: %v", err)
	}

	// Test size after deletion
	if size := idx.Size(); size != len(vectors)-1 {
		t.Errorf("Expected size %d after deletion, got %d", len(vectors)-1, size)
	}

	// Test search after deletion
	resultsAfterDelete, err := idx.Search(query, k, nil, &models.SearchParams{})
	if err != nil {
		t.Errorf("Error searching after deletion: %v", err)
	}

	// v1 should no longer be in the results
	for _, res := range resultsAfterDelete {
		if res.ID == "v1" {
			t.Errorf("Deleted vector v1 still present in search results")
		}
	}
}

func TestDifferentMetrics(t *testing.T) {
	// Create test vectors
	dim := 3
	vectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0, 0}, nil),
		models.NewVector("v2", []float32{0, 1, 0}, nil),
		models.NewVector("v3", []float32{0, 0, 1}, nil),
		models.NewVector("v4", []float32{0.5, 0.5, 0}, nil),
	}

	// Test query
	query := []float32{0.7, 0.7, 0}

	// Test each metric
	metrics := []struct {
		metric   models.DistanceMetric
		expected string // ID of the vector expected to be the closest match
	}{
		{models.Cosine, "v4"},     // Cosine: v4 has the most similar direction
		{models.DotProduct, "v4"}, // Dot product: v4 has highest dot product with query
		{models.Euclidean, "v4"},  // Euclidean: v4 is closest in Euclidean space
		{models.Manhattan, "v4"},  // Manhattan: v4 is closest in Manhattan distance
	}

	for _, tc := range metrics {
		t.Run(tc.metric.String(), func(t *testing.T) {
			// Create index with specific metric
			idx, err := NewLinearIndex(dim, tc.metric)
			if err != nil {
				t.Fatalf("Failed to create index: %v", err)
			}

			// Insert vectors
			for _, v := range vectors {
				if err := idx.Insert(v); err != nil {
					t.Errorf("Error inserting vector: %v", err)
				}
			}

			// Search
			results, err := idx.Search(query, 1, nil, &models.SearchParams{})
			if err != nil {
				t.Errorf("Error searching: %v", err)
			}

			// Check closest match
			if len(results) > 0 {
				if results[0].ID != tc.expected {
					t.Errorf("Expected closest match %s for metric %s, got %s", 
						tc.expected, tc.metric.String(), results[0].ID)
				}
			} else {
				t.Errorf("No search results returned for metric %s", tc.metric.String())
			}
		})
	}
}

func TestSearchParams(t *testing.T) {
	// Create test index
	dim := 2
	idx, err := NewLinearIndex(dim, models.Cosine)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Insert test vectors
	vectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0}, nil),
		models.NewVector("v2", []float32{0, 1}, nil),
		models.NewVector("v3", []float32{0.7, 0.7}, nil),
		models.NewVector("v4", []float32{-1, 0}, nil),
	}

	for _, v := range vectors {
		if err := idx.Insert(v); err != nil {
			t.Errorf("Error inserting vector: %v", err)
		}
	}

	// Test query
	query := []float32{0.8, 0.6}

	// Test score threshold
	t.Run("ScoreThreshold", func(t *testing.T) {
		params := &models.SearchParams{
			ScoreThreshold: 0.9, // High threshold
		}

		results, err := idx.Search(query, 10, nil, params)
		if err != nil {
			t.Errorf("Error searching: %v", err)
		}

		// With high threshold, only vectors very similar to the query should be returned
		for _, res := range results {
			if res.Score < params.ScoreThreshold {
				t.Errorf("Result %s has score %f, below threshold %f", 
					res.ID, res.Score, params.ScoreThreshold)
			}
		}

		// Check if the right vectors are filtered
		ids := make(map[string]bool)
		for _, res := range results {
			ids[res.ID] = true
		}

		// v3 should definitely be present (very similar to query)
		if !ids["v3"] {
			t.Errorf("Expected v3 to be present with high similarity")
		}

		// v4 should definitely be filtered (opposite direction)
		if ids["v4"] {
			t.Errorf("Expected v4 to be filtered out with low similarity")
		}
	})
}

func BenchmarkLinearSearch(b *testing.B) {
	// Create test vectors and index
	dim := 128
	numVectors := 1000

	idx, _ := NewLinearIndex(dim, models.Cosine)

	// Generate random vectors
	for i := 0; i < numVectors; i++ {
		values := make([]float32, dim)
		for j := 0; j < dim; j++ {
			values[j] = float32(j % 10) / 10.0 // Some pattern to make it deterministic
		}
		idx.Insert(models.NewVector(string(rune(i)), values, nil))
	}

	// Query vector
	query := make([]float32, dim)
	for i := 0; i < dim; i++ {
		query[i] = float32(i % 10) / 10.0
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Benchmark search operation
	for i := 0; i < b.N; i++ {
		idx.Search(query, 10, nil, &models.SearchParams{})
	}
}