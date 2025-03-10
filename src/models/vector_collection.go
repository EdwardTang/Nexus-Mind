package models

import (
	"fmt"
	"sync"
	"time"
)

// VectorCollection manages vectors with the same dimensionality
type VectorCollection struct {
	Name         string                // Collection name (unique identifier)
	Dimension    int                   // Fixed dimension for all vectors in this collection
	DistanceFunc DistanceMetric        // Default distance metric
	Indexes      map[string]VectorIndex // Multiple indexes for different vector fields
	MetadataSchema *MetadataSchema     // Optional schema for metadata validation
	
	// Collection-level settings
	CreatedAt    int64                 // Creation timestamp
	UpdatedAt    int64                 // Last update timestamp
	
	// Operational fields (not serialized)
	mu           sync.RWMutex          // For thread safety
}

// VectorIndex represents an interface for vector indexing structures
type VectorIndex interface {
	// Basic operations
	Insert(vector *Vector) error
	Search(query []float32, k int, filter *MetadataFilter, params *SearchParams) ([]SearchResult, error)
	Delete(id string) error
	BatchInsert(vectors []*Vector) error
	
	// Statistics and info
	Size() int
	Dimension() int
	
	// Persistence
	Load() error
	Save() error
}

// DistanceMetric defines different ways to measure vector similarity
type DistanceMetric int

const (
	Cosine DistanceMetric = iota    // Cosine similarity
	DotProduct                      // Dot product
	Euclidean                       // Euclidean distance
	Manhattan                       // Manhattan distance
)

// String returns the name of the distance metric
func (d DistanceMetric) String() string {
	switch d {
	case Cosine:
		return "Cosine"
	case DotProduct:
		return "DotProduct"
	case Euclidean:
		return "Euclidean"
	case Manhattan:
		return "Manhattan"
	default:
		return "Unknown"
	}
}

// SearchResult represents a single search result
type SearchResult struct {
	ID       string    // Vector ID
	Distance float32   // Distance/similarity score
	Vector   *Vector   // Optional vector data (may be nil if not requested)
	Score    float32   // Normalized score (1.0 = best match, 0.0 = worst)
}

// SearchParams controls how vector search is performed
type SearchParams struct {
	// HNSW specific parameters
	HnswEf          int     // Size of the dynamic candidate list
	
	// General search configuration
	Exact           bool    // Whether to use exact search (bypassing indexes)
	IndexedOnly     bool    // Search only in indexed segments
	UseQuantization bool    // Whether to use vector quantization for faster search
	
	// Search strategy
	SearchStrategy  SearchStrategy
	
	// Result filtering
	ScoreThreshold  float32 // Minimum score threshold for results
}

// SearchStrategy determines algorithm behavior during search
type SearchStrategy int

const (
	Default SearchStrategy = iota
	ExactSearch            // Brute force, no index
	FastSearch             // Optimize for speed over accuracy
	PreciseSearch          // Optimize for accuracy over speed
	BatchSearch            // Optimize for throughput of multiple queries
)

// NewSearchParams creates default search parameters
func NewSearchParams() *SearchParams {
	return &SearchParams{
		HnswEf:         100,   // Default HNSW ef
		Exact:          false, // Use index by default
		IndexedOnly:    false, // Include all segments
		SearchStrategy: Default,
	}
}

// NewFastSearchParams creates parameters optimized for speed
func NewFastSearchParams() *SearchParams {
	return &SearchParams{
		HnswEf:         40,        // Lower ef for faster search
		SearchStrategy: FastSearch,
	}
}

// NewPreciseSearchParams creates parameters optimized for accuracy
func NewPreciseSearchParams() *SearchParams {
	return &SearchParams{
		HnswEf:         300,          // Higher ef for more accurate search
		SearchStrategy: PreciseSearch,
	}
}

// NewVectorCollection creates a new collection for vectors
func NewVectorCollection(name string, dimension int, distanceMetric DistanceMetric) *VectorCollection {
	now := time.Now().UnixNano()
	return &VectorCollection{
		Name:          name,
		Dimension:     dimension,
		DistanceFunc:  distanceMetric,
		Indexes:       make(map[string]VectorIndex),
		MetadataSchema: NewMetadataSchema(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// AddIndex adds a new index to the collection
func (c *VectorCollection) AddIndex(name string, index VectorIndex) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if index.Dimension() != c.Dimension {
		return fmt.Errorf("index dimension %d does not match collection dimension %d", 
			index.Dimension(), c.Dimension)
	}
	
	c.Indexes[name] = index
	c.UpdatedAt = time.Now().UnixNano()
	return nil
}

// Insert adds a vector to the collection
func (c *VectorCollection) Insert(vector *Vector) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Validate vector dimension
	if len(vector.Values) != c.Dimension {
		return fmt.Errorf("vector dimension %d does not match collection dimension %d",
			len(vector.Values), c.Dimension)
	}
	
	// Validate metadata if schema is defined
	if c.MetadataSchema != nil && len(c.MetadataSchema.Fields) > 0 {
		if err := c.MetadataSchema.ValidateMetadata(vector.Metadata); err != nil {
			return err
		}
	}
	
	// Add to all indexes
	for name, index := range c.Indexes {
		if err := index.Insert(vector); err != nil {
			return fmt.Errorf("failed to insert into index %s: %w", name, err)
		}
	}
	
	c.UpdatedAt = time.Now().UnixNano()
	return nil
}

// BatchInsert adds multiple vectors at once
func (c *VectorCollection) BatchInsert(vectors []*Vector) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Validate all vectors first
	for i, vector := range vectors {
		// Validate vector dimension
		if len(vector.Values) != c.Dimension {
			return fmt.Errorf("vector %d: dimension %d does not match collection dimension %d",
				i, len(vector.Values), c.Dimension)
		}
		
		// Validate metadata if schema is defined
		if c.MetadataSchema != nil && len(c.MetadataSchema.Fields) > 0 {
			if err := c.MetadataSchema.ValidateMetadata(vector.Metadata); err != nil {
				return fmt.Errorf("vector %d: %w", i, err)
			}
		}
	}
	
	// Insert into all indexes
	for name, index := range c.Indexes {
		if err := index.BatchInsert(vectors); err != nil {
			return fmt.Errorf("failed to batch insert into index %s: %w", name, err)
		}
	}
	
	c.UpdatedAt = time.Now().UnixNano()
	return nil
}

// Delete removes a vector from the collection
func (c *VectorCollection) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Delete from all indexes
	for name, index := range c.Indexes {
		if err := index.Delete(id); err != nil {
			return fmt.Errorf("failed to delete from index %s: %w", name, err)
		}
	}
	
	c.UpdatedAt = time.Now().UnixNano()
	return nil
}

// Search performs a vector similarity search
func (c *VectorCollection) Search(
	query []float32, 
	k int, 
	filter *MetadataFilter, 
	params *SearchParams,
) ([]SearchResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Validate query dimension
	if len(query) != c.Dimension {
		return nil, fmt.Errorf("query dimension %d does not match collection dimension %d",
			len(query), c.Dimension)
	}
	
	// Use default params if not provided
	if params == nil {
		params = NewSearchParams()
	}
	
	// Choose the most appropriate index
	// For a more sophisticated implementation, we would have a query planner here
	if len(c.Indexes) == 0 {
		return nil, fmt.Errorf("no indexes available in collection %s", c.Name)
	}
	
	// For now, just use the first index
	// In a real implementation, we would choose based on the search strategy
	for _, index := range c.Indexes {
		return index.Search(query, k, filter, params)
	}
	
	// This should never happen as we check for empty indexes above
	return nil, fmt.Errorf("no index selected for search")
}

// Query performs a universal query against the collection
// This implements the flexible Query API described in the design document
func (c *VectorCollection) Query(request *QueryRequest) (interface{}, error) {
	// This is just a stub - full implementation would be more complex
	// and would handle all the different query types
	
	// For now, just implement vector search
	if request.Vector != nil {
		return c.Search(
			request.Vector, 
			request.Limit, 
			request.Filter, 
			request.Params,
		)
	}
	
	return nil, fmt.Errorf("unsupported query type")
}

// Size returns the number of vectors in the collection
func (c *VectorCollection) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Sum size from all indexes
	// This is a simplification - in reality vectors might be in multiple indexes
	if len(c.Indexes) == 0 {
		return 0
	}
	
	// Just return the size of the first index
	for _, index := range c.Indexes {
		return index.Size()
	}
	
	return 0
}

// QueryRequest represents a universal query request
// This is the implementation of the unified Query API from the design document
type QueryRequest struct {
	// One of the following must be specified
	Vector       []float32         // Vector search (kNN)
	PointID      string            // Search by existing point ID
	Recommend    *RecommendParams  // Recommendation by examples
	Scroll       *ScrollParams     // Pagination through all points
	Sample       string            // Random sampling ("random")
	
	// Optional parameters
	Filter       *MetadataFilter   // Filtering conditions
	Params       *SearchParams     // Search behavior configuration
	Limit        int               // Maximum results to return
	Offset       int               // Number of results to skip
	WithVectors  bool              // Include vectors in response
	WithPayload  interface{}       // Control payload inclusion
	
	// Grouping parameters
	GroupBy      string            // Field to group results by
	GroupSize    int               // Maximum points per group
	GroupLimit   int               // Maximum groups to return
	
	// For multi-vector collections
	Using        string            // Which vector field to use
}

// RecommendParams controls recommendation behavior
type RecommendParams struct {
	Positive []string  // IDs of positive examples
	Negative []string  // IDs of negative examples
	Strategy string    // Recommendation strategy (average, weighted, etc.)
}

// ScrollParams controls scrolling through all vectors
type ScrollParams struct {
	Offset string    // Pagination cursor
	Limit  int       // Number of results per page
}