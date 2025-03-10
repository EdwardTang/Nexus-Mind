package query

import (
	"course/models"
	"reflect"
	"testing"
)

// Mock vector collection for testing
type MockVectorCollection struct {
	name      string
	dimension int
	vectors   map[string]*models.Vector
	schema    *models.MetadataSchema
}

func NewMockCollection(name string, dim int) *MockVectorCollection {
	return &MockVectorCollection{
		name:      name,
		dimension: dim,
		vectors:   make(map[string]*models.Vector),
	}
}

func (m *MockVectorCollection) Name() string {
	return m.name
}

func (m *MockVectorCollection) Dimension() int {
	return m.dimension
}

func (m *MockVectorCollection) Insert(vector *models.Vector) error {
	m.vectors[vector.ID] = vector
	return nil
}

func (m *MockVectorCollection) Get(id string) (*models.Vector, bool) {
	v, exists := m.vectors[id]
	return v, exists
}

func (m *MockVectorCollection) Delete(id string) error {
	delete(m.vectors, id)
	return nil
}

func (m *MockVectorCollection) Size() int {
	return len(m.vectors)
}

func (m *MockVectorCollection) Search(req *models.QueryRequest) ([]*models.SearchResult, error) {
	results := make([]*models.SearchResult, 0)
	
	// Simple mock implementation
	for id, vector := range m.vectors {
		// Include only vectors that match the filter
		if req.Filter == nil || req.Filter.Matches(vector.Metadata) {
			results = append(results, &models.SearchResult{
				ID:     id,
				Score:  0.9, // Fixed mock score
				Vector: vector,
			})
		}
	}
	
	// Limit results
	if len(results) > req.TopK {
		results = results[:req.TopK]
	}
	
	return results, nil
}

func (m *MockVectorCollection) SetSchema(schema *models.MetadataSchema) {
	m.schema = schema
}

func (m *MockVectorCollection) GetSchema() *models.MetadataSchema {
	return m.schema
}

func TestProcessorCreation(t *testing.T) {
	// Create processor
	processor := NewProcessor()
	
	if processor == nil {
		t.Fatalf("Failed to create processor")
	}
	
	// Should start with no collections
	if len(processor.collections) != 0 {
		t.Errorf("New processor should have 0 collections, got %d", len(processor.collections))
	}
}

func TestCollectionManagement(t *testing.T) {
	processor := NewProcessor()
	
	// Create mock collections
	coll1 := NewMockCollection("test1", 4)
	coll2 := NewMockCollection("test2", 8)
	
	// Register collections
	processor.RegisterCollection("test1", coll1)
	processor.RegisterCollection("test2", coll2)
	
	// Check collections count
	if len(processor.collections) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(processor.collections))
	}
	
	// Get collection and verify
	retrieved, err := processor.GetCollection("test1")
	if err != nil {
		t.Errorf("Failed to get collection: %v", err)
	}
	
	if retrieved != coll1 {
		t.Errorf("Retrieved collection is not the same as registered")
	}
	
	// Try getting non-existent collection
	_, err = processor.GetCollection("nonexistent")
	if err == nil {
		t.Errorf("Expected error when getting non-existent collection, got nil")
	}
	
	// Remove collection
	err = processor.RemoveCollection("test1")
	if err != nil {
		t.Errorf("Failed to remove collection: %v", err)
	}
	
	// Check collections count after removal
	if len(processor.collections) != 1 {
		t.Errorf("Expected 1 collection after removal, got %d", len(processor.collections))
	}
	
	// Verify the collection is gone
	_, err = processor.GetCollection("test1")
	if err == nil {
		t.Errorf("Expected error when getting removed collection, got nil")
	}
	
	// Try removing non-existent collection
	err = processor.RemoveCollection("nonexistent")
	if err == nil {
		t.Errorf("Expected error when removing non-existent collection, got nil")
	}
}

func TestSimilaritySearch(t *testing.T) {
	processor := NewProcessor()
	
	// Create mock collection
	coll := NewMockCollection("test", 3)
	processor.RegisterCollection("test", coll)
	
	// Add test vectors
	vectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0, 0}, map[string]interface{}{"category": "A"}),
		models.NewVector("v2", []float32{0, 1, 0}, map[string]interface{}{"category": "B"}),
		models.NewVector("v3", []float32{0, 0, 1}, map[string]interface{}{"category": "A"}),
	}
	
	for _, v := range vectors {
		coll.Insert(v)
	}
	
	// Test similarity search
	query := []float32{1, 0, 0}
	k := 2
	
	// Create search request
	req := &SearchRequest{
		CollectionName: "test",
		QueryVector:    query,
		TopK:           k,
		Filter:         nil,
		Strategy:       "default",
	}
	
	// Execute search
	resp, err := processor.Search(req)
	if err != nil {
		t.Errorf("Search error: %v", err)
	}
	
	// Check results
	if len(resp.Results) != k {
		t.Errorf("Expected %d results, got %d", k, len(resp.Results))
	}
	
	// Test with filter
	filter := models.NewEqualsCondition("category", "A")
	filteredReq := &SearchRequest{
		CollectionName: "test",
		QueryVector:    query,
		TopK:           10, // Request more than available to get all matches
		Filter:         filter,
		Strategy:       "default",
	}
	
	filteredResp, err := processor.Search(filteredReq)
	if err != nil {
		t.Errorf("Filtered search error: %v", err)
	}
	
	// Check that all results match the filter
	for _, result := range filteredResp.Results {
		if result.Vector.Metadata["category"] != "A" {
			t.Errorf("Filter failed: result %s has category %v, expected A", 
				result.ID, result.Vector.Metadata["category"])
		}
	}
	
	// Test with non-existent collection
	badReq := &SearchRequest{
		CollectionName: "nonexistent",
		QueryVector:    query,
		TopK:           k,
	}
	
	_, err = processor.Search(badReq)
	if err == nil {
		t.Errorf("Expected error when searching in non-existent collection, got nil")
	}
}

func TestSearchStrategies(t *testing.T) {
	processor := NewProcessor()
	
	// Create mock collection
	coll := NewMockCollection("test", 3)
	processor.RegisterCollection("test", coll)
	
	// Add test vectors
	vectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0, 0}, nil),
		models.NewVector("v2", []float32{0, 1, 0}, nil),
		models.NewVector("v3", []float32{0, 0, 1}, nil),
	}
	
	for _, v := range vectors {
		coll.Insert(v)
	}
	
	// Query vector
	query := []float32{1, 0, 0}
	
	// Test each strategy
	strategies := []string{
		"default",
		"exact",
		"fast",
		"precise",
	}
	
	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			req := &SearchRequest{
				CollectionName: "test",
				QueryVector:    query,
				TopK:           2,
				Strategy:       strategy,
			}
			
			resp, err := processor.Search(req)
			if err != nil {
				t.Errorf("Search with strategy %s error: %v", strategy, err)
			}
			
			if len(resp.Results) != 2 {
				t.Errorf("Strategy %s: Expected 2 results, got %d", 
					strategy, len(resp.Results))
			}
		})
	}
	
	// Test invalid strategy
	invalidReq := &SearchRequest{
		CollectionName: "test",
		QueryVector:    query,
		TopK:           2,
		Strategy:       "invalid_strategy",
	}
	
	_, err := processor.Search(invalidReq)
	if err == nil {
		t.Errorf("Expected error for invalid strategy, got nil")
	}
}

func TestVectorOperations(t *testing.T) {
	processor := NewProcessor()
	
	// Create mock collection
	coll := NewMockCollection("test", 3)
	processor.RegisterCollection("test", coll)
	
	// Test vector insertion
	vectorReq := &VectorRequest{
		CollectionName: "test",
		Vector: &models.Vector{
			ID:     "test_vector",
			Values: []float32{1, 2, 3},
			Metadata: map[string]interface{}{
				"key": "value",
			},
		},
	}
	
	err := processor.InsertVector(vectorReq)
	if err != nil {
		t.Errorf("InsertVector error: %v", err)
	}
	
	// Verify vector was inserted
	if size := coll.Size(); size != 1 {
		t.Errorf("Expected collection size 1 after insert, got %d", size)
	}
	
	// Get the vector
	getReq := &VectorRequest{
		CollectionName: "test",
		Vector: &models.Vector{
			ID: "test_vector",
		},
	}
	
	vector, err := processor.GetVector(getReq)
	if err != nil {
		t.Errorf("GetVector error: %v", err)
	}
	
	if vector.ID != "test_vector" {
		t.Errorf("Expected ID test_vector, got %s", vector.ID)
	}
	
	// Check values
	expectedValues := []float32{1, 2, 3}
	if !reflect.DeepEqual(vector.Values, expectedValues) {
		t.Errorf("Expected values %v, got %v", expectedValues, vector.Values)
	}
	
	// Delete the vector
	deleteReq := &VectorRequest{
		CollectionName: "test",
		Vector: &models.Vector{
			ID: "test_vector",
		},
	}
	
	err = processor.DeleteVector(deleteReq)
	if err != nil {
		t.Errorf("DeleteVector error: %v", err)
	}
	
	// Verify vector was deleted
	if size := coll.Size(); size != 0 {
		t.Errorf("Expected collection size 0 after delete, got %d", size)
	}
	
	// Try getting the deleted vector
	_, err = processor.GetVector(getReq)
	if err == nil {
		t.Errorf("Expected error when getting deleted vector, got nil")
	}
	
	// Test with non-existent collection
	badReq := &VectorRequest{
		CollectionName: "nonexistent",
		Vector: &models.Vector{
			ID: "test_vector",
		},
	}
	
	err = processor.InsertVector(badReq)
	if err == nil {
		t.Errorf("Expected error when inserting to non-existent collection, got nil")
	}
}