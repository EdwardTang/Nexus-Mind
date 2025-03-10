package index

import (
	"fmt"
	"sort"
	"sync"

	"course/models"
	"course/vector"
)

// LinearIndex is a simple brute-force index that performs linear (exhaustive) search
// It's not efficient for large datasets but is useful as a baseline and for testing
type LinearIndex struct {
	dimension     int
	distanceFunc  vector.DistanceFunc
	metric        models.DistanceMetric
	vectors       map[string]*models.Vector
	keepNormalized bool
	mu            sync.RWMutex
}

// NewLinearIndex creates a new brute-force search index
func NewLinearIndex(dimension int, metric models.DistanceMetric) (*LinearIndex, error) {
	distFunc, err := vector.GetDistanceFunc(metric)
	if err != nil {
		return nil, err
	}

	return &LinearIndex{
		dimension:     dimension,
		distanceFunc:  distFunc,
		metric:        metric,
		vectors:       make(map[string]*models.Vector),
		keepNormalized: metric == models.Cosine, // Precompute normalization for cosine
	}, nil
}

// Insert adds a vector to the index
func (idx *LinearIndex) Insert(v *models.Vector) error {
	if len(v.Values) != idx.dimension {
		return fmt.Errorf("vector dimension %d does not match index dimension %d", 
			len(v.Values), idx.dimension)
	}

	// Create a copy to avoid external modifications
	vectorCopy := v.Copy()
	
	// Normalize if needed (for cosine similarity)
	if idx.keepNormalized {
		vectorCopy.Normalize()
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()
	
	idx.vectors[v.ID] = vectorCopy
	return nil
}

// Search performs a brute-force search for the nearest neighbors
func (idx *LinearIndex) Search(
	query []float32, 
	k int, 
	filter *models.MetadataFilter, 
	params *models.SearchParams,
) ([]models.SearchResult, error) {
	if len(query) != idx.dimension {
		return nil, fmt.Errorf("query dimension %d does not match index dimension %d",
			len(query), idx.dimension)
	}

	// Normalize the query if needed
	queryCopy := make([]float32, len(query))
	copy(queryCopy, query)
	
	if idx.keepNormalized {
		vector.NormalizeVector(queryCopy)
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Calculate the number of results we need
	if k <= 0 {
		k = 10 // Default to 10 results
	}
	if k > len(idx.vectors) {
		k = len(idx.vectors)
	}

	// Calculate distances for all vectors
	results := make([]models.SearchResult, 0, len(idx.vectors))
	var scoreThreshold float32 = -1
	if params != nil && params.ScoreThreshold > 0 {
		scoreThreshold = params.ScoreThreshold
	}

	// We use a channel to process vectors in parallel
	type distanceResult struct {
		id       string
		vector   *models.Vector
		distance float32
	}

	// Choose how many goroutines to use
	numWorkers := 4
	if len(idx.vectors) < 1000 {
		numWorkers = 1 // Use single-threaded for small datasets
	}

	// Create a channel for distributing work and collecting results
	workCh := make(chan *models.Vector, numWorkers)
	resultCh := make(chan distanceResult, numWorkers)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for vec := range workCh {
				// Skip deleted vectors
				if vec.Deleted {
					continue
				}

				// Apply filter if provided
				if filter != nil && !filter.MatchVector(vec) {
					continue
				}

				// Calculate distance
				distance := idx.distanceFunc(queryCopy, vec.Values)
				
				resultCh <- distanceResult{
					id:       vec.ID,
					vector:   vec,
					distance: distance,
				}
			}
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Feed vectors to workers
	go func() {
		for _, vec := range idx.vectors {
			workCh <- vec
		}
		close(workCh)
	}()

	// Collect results
	for res := range resultCh {
		score := vector.NormalizeScore(res.distance, idx.metric)
		
		// Apply score threshold if provided
		if scoreThreshold > 0 && score < scoreThreshold {
			continue
		}
		
		results = append(results, models.SearchResult{
			ID:       res.id,
			Distance: res.distance,
			Vector:   res.vector,
			Score:    score,
		})
	}

	// Sort the results
	if vector.IsHigherBetter(idx.metric) {
		// Sort by distance in descending order for similarity metrics
		sort.Slice(results, func(i, j int) bool {
			return results[i].Distance > results[j].Distance
		})
	} else {
		// Sort by distance in ascending order for distance metrics
		sort.Slice(results, func(i, j int) bool {
			return results[i].Distance < results[j].Distance
		})
	}

	// Return the top k results
	if len(results) > k {
		results = results[:k]
	}

	return results, nil
}

// Delete removes a vector from the index
func (idx *LinearIndex) Delete(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	
	if vec, exists := idx.vectors[id]; exists {
		// Mark as deleted (soft deletion)
		vec.MarkDeleted()
		return nil
	}
	
	// Alternatively, we could do a hard deletion
	// delete(idx.vectors, id)
	
	return fmt.Errorf("vector with ID %s not found", id)
}

// BatchInsert adds multiple vectors to the index
func (idx *LinearIndex) BatchInsert(vectors []*models.Vector) error {
	for _, v := range vectors {
		if err := idx.Insert(v); err != nil {
			return err
		}
	}
	return nil
}

// Size returns the number of vectors in the index
func (idx *LinearIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	// Count non-deleted vectors
	count := 0
	for _, vec := range idx.vectors {
		if !vec.Deleted {
			count++
		}
	}
	
	return count
}

// Dimension returns the dimensionality of the index
func (idx *LinearIndex) Dimension() int {
	return idx.dimension
}

// Load loads the index from disk (stub implementation)
func (idx *LinearIndex) Load() error {
	// Not implemented for the linear index
	return nil
}

// Save saves the index to disk (stub implementation)
func (idx *LinearIndex) Save() error {
	// Not implemented for the linear index
	return nil
}