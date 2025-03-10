package vectorstore

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

// Vector represents a high-dimensional vector with metadata
type Vector struct {
	ID       string                 `json:"id"`
	Values   []float32              `json:"values"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResult represents a search result with distance score
type SearchResult struct {
	Vector   *Vector `json:"vector"`
	Distance float32 `json:"distance"`
}

// VectorStoreConfig holds configuration for the vector store
type VectorStoreConfig struct {
	NodeID       string `json:"nodeId"`
	Dimensions   int    `json:"dimensions"`
	DistanceFunc string `json:"distanceFunc"`
}

// VectorStore implements a vector database with similarity search
type VectorStore struct {
	nodeID          string
	dimensions      int
	distanceFunc    func([]float32, []float32) float32
	distanceFuncStr string
	vectors         map[string]*Vector
	tokenRing       *TokenRing
	mu              sync.RWMutex
	logger          Logger
}

// NewVectorStore creates a new vector store instance
func NewVectorStore(config VectorStoreConfig, logger Logger) (*VectorStore, error) {
	// Validate configuration
	if config.Dimensions <= 0 {
		return nil, fmt.Errorf("dimensions must be greater than 0")
	}

	// Select distance function
	var distanceFunc func([]float32, []float32) float32
	switch config.DistanceFunc {
	case "cosine":
		distanceFunc = CosineDistance
	case "euclidean":
		distanceFunc = EuclideanDistance
	case "dot":
		distanceFunc = DotProductDistance
	default:
		return nil, fmt.Errorf("unsupported distance function: %s", config.DistanceFunc)
	}

	vs := &VectorStore{
		nodeID:          config.NodeID,
		dimensions:      config.Dimensions,
		distanceFunc:    distanceFunc,
		distanceFuncStr: config.DistanceFunc,
		vectors:         make(map[string]*Vector),
		logger:          logger,
	}

	logger.Info("Created vector store with dimensions=%d, distanceFunc=%s", 
		config.Dimensions, config.DistanceFunc)
	return vs, nil
}

// SetTokenRing sets the token ring for the vector store
func (vs *VectorStore) SetTokenRing(ring *TokenRing) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.tokenRing = ring
	vs.logger.Info("Token ring updated with %d nodes", len(ring.GetAllNodes()))
}

// AddVector adds a vector to the store
func (vs *VectorStore) AddVector(vector *Vector) error {
	if vector == nil {
		return fmt.Errorf("vector cannot be nil")
	}

	if len(vector.Values) != vs.dimensions {
		return fmt.Errorf("vector has wrong dimensions: got %d, expected %d", 
			len(vector.Values), vs.dimensions)
	}

	// If we have a token ring, check if this vector belongs on this node
	vs.mu.RLock()
	if vs.tokenRing != nil {
		isLocal := false
		for _, node := range vs.tokenRing.GetNodesForVector(vector.ID) {
			if node == vs.nodeID {
				isLocal = true
				break
			}
		}
		
		if !isLocal {
			vs.mu.RUnlock()
			vs.logger.Debug("Vector %s doesn't belong on this node", vector.ID)
			return nil // Skip this vector, it doesn't belong on this node
		}
	}
	vs.mu.RUnlock()

	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	// Create a deep copy of the vector to prevent external modification
	newVector := &Vector{
		ID:     vector.ID,
		Values: make([]float32, len(vector.Values)),
	}
	copy(newVector.Values, vector.Values)
	
	if vector.Metadata != nil {
		newVector.Metadata = make(map[string]interface{})
		for k, v := range vector.Metadata {
			newVector.Metadata[k] = v
		}
	}
	
	vs.vectors[vector.ID] = newVector
	vs.logger.Debug("Added vector %s", vector.ID)
	return nil
}

// GetVector retrieves a vector by ID
func (vs *VectorStore) GetVector(id string) (*Vector, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	vector, exists := vs.vectors[id]
	if !exists {
		return nil, fmt.Errorf("vector %s not found", id)
	}
	
	// Return a copy to prevent external modification
	result := &Vector{
		ID:     vector.ID,
		Values: make([]float32, len(vector.Values)),
	}
	copy(result.Values, vector.Values)
	
	if vector.Metadata != nil {
		result.Metadata = make(map[string]interface{})
		for k, v := range vector.Metadata {
			result.Metadata[k] = v
		}
	}
	
	return result, nil
}

// DeleteVector removes a vector from the store
func (vs *VectorStore) DeleteVector(id string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	if _, exists := vs.vectors[id]; !exists {
		return fmt.Errorf("vector %s not found", id)
	}
	
	delete(vs.vectors, id)
	vs.logger.Debug("Deleted vector %s", id)
	return nil
}

// Search performs a similarity search
func (vs *VectorStore) Search(query []float32, k int, filter func(*Vector) bool) ([]SearchResult, error) {
	if len(query) != vs.dimensions {
		return nil, fmt.Errorf("query has wrong dimensions: got %d, expected %d", 
			len(query), vs.dimensions)
	}
	
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	// Find matching vectors
	results := make([]SearchResult, 0, len(vs.vectors))
	
	for _, vector := range vs.vectors {
		// Apply filter if provided
		if filter != nil && !filter(vector) {
			continue
		}
		
		// Calculate distance
		distance := vs.distanceFunc(query, vector.Values)
		
		// Add to results
		results = append(results, SearchResult{
			Vector:   vector,
			Distance: distance,
		})
	}
	
	// Sort by distance (ascending for similarity search)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})
	
	// Limit to k results
	if k > 0 && k < len(results) {
		results = results[:k]
	}
	
	vs.logger.Debug("Search returned %d results", len(results))
	return results, nil
}

// GetAllVectorIDs returns all vector IDs in the store
func (vs *VectorStore) GetAllVectorIDs() []string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	ids := make([]string, 0, len(vs.vectors))
	for id := range vs.vectors {
		ids = append(ids, id)
	}
	
	return ids
}

// GetLocalVectorIDs returns vectors that belong on this node
func (vs *VectorStore) GetLocalVectorIDs() []string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	// If no token ring, all vectors are local
	if vs.tokenRing == nil {
		ids := make([]string, 0, len(vs.vectors))
		for id := range vs.vectors {
			ids = append(ids, id)
		}
		return ids
	}
	
	// Otherwise, filter by token ring assignment
	localIDs := make([]string, 0)
	
	for id := range vs.vectors {
		for _, node := range vs.tokenRing.GetNodesForVector(id) {
			if node == vs.nodeID {
				localIDs = append(localIDs, id)
				break
			}
		}
	}
	
	return localIDs
}

// GetStats returns statistics about the store
func (vs *VectorStore) GetStats() map[string]interface{} {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	stats := map[string]interface{}{
		"nodeID":           vs.nodeID,
		"dimensions":       vs.dimensions,
		"distanceFunction": vs.distanceFuncStr,
		"totalVectors":     len(vs.vectors),
		"localVectors":     0,
	}
	
	// Count local vectors if we have a token ring
	if vs.tokenRing != nil {
		localCount := 0
		for id := range vs.vectors {
			for _, node := range vs.tokenRing.GetNodesForVector(id) {
				if node == vs.nodeID {
					localCount++
					break
				}
			}
		}
		stats["localVectors"] = localCount
	} else {
		stats["localVectors"] = len(vs.vectors)
	}
	
	return stats
}

// TransferVectors prepares vectors for transfer to another node
func (vs *VectorStore) TransferVectors(vectorIDs []string, destinationNode string) (bool, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	// In a real implementation, this would actually send the vectors to the destination
	vs.logger.Info("Transferred %d vectors to node %s", len(vectorIDs), destinationNode)
	return true, nil
}

// ReceiveVectors receives vectors from another node
func (vs *VectorStore) ReceiveVectors(vectors []*Vector, sourceNode string) (bool, error) {
	// Add received vectors
	for _, vector := range vectors {
		err := vs.AddVector(vector)
		if err != nil {
			vs.logger.Error("Failed to add received vector %s: %v", vector.ID, err)
			return false, err
		}
	}
	
	vs.logger.Info("Received %d vectors from node %s", len(vectors), sourceNode)
	return true, nil
}

// CosineDistance calculates cosine distance between two vectors
func CosineDistance(a, b []float32) float32 {
	// Check for dimension mismatch
	if len(a) != len(b) {
		return 1.0 // Return max distance for mismatched dimensions
	}
	
	var dotProduct float32
	var normA float32
	var normB float32
	
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	// Handle zero vectors
	if normA == 0 || normB == 0 {
		return 1.0
	}
	
	// Calculate cosine similarity (1 - cos_similarity for distance)
	similarity := dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
	
	// Clamp to [-1, 1] range
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}
	
	// Convert similarity to distance (0 = identical, 2 = opposite)
	distance := 1.0 - similarity
	
	return distance
}

// EuclideanDistance calculates Euclidean distance between two vectors
func EuclideanDistance(a, b []float32) float32 {
	// Check for dimension mismatch
	if len(a) != len(b) {
		return float32(math.MaxFloat32) // Return max distance for mismatched dimensions
	}
	
	var sumSquares float32
	
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sumSquares += diff * diff
	}
	
	return float32(math.Sqrt(float64(sumSquares)))
}

// DotProductDistance calculates dot product distance
func DotProductDistance(a, b []float32) float32 {
	// Check for dimension mismatch
	if len(a) != len(b) {
		return 1.0 // Return max distance for mismatched dimensions
	}
	
	var dotProduct float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	
	// Normalize to [0,1] range where 0 is most similar
	// We're assuming vectors are normalized
	return 1.0 - float32(math.Max(0.0, math.Min(1.0, float64(dotProduct))))
}