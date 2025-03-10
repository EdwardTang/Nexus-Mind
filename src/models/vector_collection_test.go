package models

import (
	"reflect"
	"testing"
)

// Mock implementation of VectorIndex for testing
type MockVectorIndex struct {
	vectors   map[string]*Vector
	dimension int
	metric    DistanceMetric
}

func NewMockIndex(dim int, metric DistanceMetric) *MockVectorIndex {
	return &MockVectorIndex{
		vectors:   make(map[string]*Vector),
		dimension: dim,
		metric:    metric,
	}
}

func (m *MockVectorIndex) Insert(vector *Vector) error {
	m.vectors[vector.ID] = vector
	return nil
}

func (m *MockVectorIndex) Delete(id string) error {
	if _, exists := m.vectors[id]; !exists {
		return ErrVectorNotFound
	}
	delete(m.vectors, id)
	return nil
}

func (m *MockVectorIndex) Search(query []float32, limit int, filter MetadataFilter, params *SearchParams) ([]*SearchResult, error) {
	results := make([]*SearchResult, 0)
	
	for _, v := range m.vectors {
		// Simple mock implementation - doesn't actually perform distance calculation
		// Just includes vectors that match the filter
		if filter == nil || filter.Matches(v.Metadata) {
			results = append(results, &SearchResult{
				ID:     v.ID,
				Score:  0.9, // Mock score
				Vector: v,
			})
		}
	}
	
	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}
	
	return results, nil
}

func (m *MockVectorIndex) Size() int {
	return len(m.vectors)
}

func (m *MockVectorIndex) Dimension() int {
	return m.dimension
}

func (m *MockVectorIndex) Get(id string) (*Vector, bool) {
	v, exists := m.vectors[id]
	return v, exists
}

func TestVectorCollectionCreate(t *testing.T) {
	// Create a collection
	name := "test_collection"
	dim := 4
	metric := Cosine
	
	collection, err := NewVectorCollection(name, dim, metric)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}
	
	// Check properties
	if collection.Name != name {
		t.Errorf("Expected name %s, got %s", name, collection.Name)
	}
	
	if collection.Dimension != dim {
		t.Errorf("Expected dimension %d, got %d", dim, collection.Dimension)
	}
	
	if collection.Metric != metric {
		t.Errorf("Expected metric %s, got %s", metric.String(), collection.Metric.String())
	}
	
	// Index should be initialized
	if collection.Index == nil {
		t.Errorf("Index not initialized")
	}
	
	// Collection should start with 0 vectors
	if size := collection.Size(); size != 0 {
		t.Errorf("Expected size 0, got %d", size)
	}
}

func TestVectorCollectionInsertDelete(t *testing.T) {
	// Create collection
	collection, _ := NewVectorCollection("test", 3, Cosine)
	
	// Replace the index with a mock for testing
	mockIndex := NewMockIndex(3, Cosine)
	collection.Index = mockIndex
	
	// Create test vectors
	vectors := []*Vector{
		NewVector("v1", []float32{1, 0, 0}, map[string]interface{}{"tag": "A"}),
		NewVector("v2", []float32{0, 1, 0}, map[string]interface{}{"tag": "B"}),
		NewVector("v3", []float32{0, 0, 1}, map[string]interface{}{"tag": "C"}),
	}
	
	// Insert vectors
	for _, v := range vectors {
		err := collection.Insert(v)
		if err != nil {
			t.Errorf("Failed to insert vector %s: %v", v.ID, err)
		}
	}
	
	// Check size
	if size := collection.Size(); size != len(vectors) {
		t.Errorf("Expected size %d, got %d", len(vectors), size)
	}
	
	// Test retrieval
	for _, v := range vectors {
		retrieved, exists := collection.Get(v.ID)
		if !exists {
			t.Errorf("Vector %s not found after insertion", v.ID)
			continue
		}
		
		if retrieved.ID != v.ID {
			t.Errorf("Retrieved ID mismatch: expected %s, got %s", v.ID, retrieved.ID)
		}
		
		if !reflect.DeepEqual(retrieved.Values, v.Values) {
			t.Errorf("Retrieved values mismatch for %s", v.ID)
		}
	}
	
	// Test deletion
	err := collection.Delete("v2")
	if err != nil {
		t.Errorf("Error deleting vector: %v", err)
	}
	
	// Check size after deletion
	if size := collection.Size(); size != len(vectors)-1 {
		t.Errorf("Expected size %d after deletion, got %d", len(vectors)-1, size)
	}
	
	// Verify deleted vector is gone
	if _, exists := collection.Get("v2"); exists {
		t.Errorf("Vector v2 still exists after deletion")
	}
	
	// Test deleting non-existent vector
	err = collection.Delete("nonexistent")
	if err == nil {
		t.Errorf("Expected error when deleting non-existent vector, got nil")
	}
}

func TestVectorCollectionSearch(t *testing.T) {
	// Create collection with mock index
	collection, _ := NewVectorCollection("test", 3, Cosine)
	mockIndex := NewMockIndex(3, Cosine)
	collection.Index = mockIndex
	
	// Insert test vectors with different tags
	vectors := []*Vector{
		NewVector("v1", []float32{1, 0, 0}, map[string]interface{}{"tag": "A"}),
		NewVector("v2", []float32{0, 1, 0}, map[string]interface{}{"tag": "B"}),
		NewVector("v3", []float32{0, 0, 1}, map[string]interface{}{"tag": "A"}),
		NewVector("v4", []float32{1, 1, 0}, map[string]interface{}{"tag": "B"}),
	}
	
	for _, v := range vectors {
		collection.Insert(v)
	}
	
	// Test basic search
	query := []float32{1, 0, 0}
	limit := 2
	
	results, err := collection.Search(&QueryRequest{
		Vector: query,
		TopK:   limit,
		Filter: nil,
		Params: &SearchParams{},
	})
	
	if err != nil {
		t.Errorf("Search error: %v", err)
	}
	
	if len(results) != limit {
		t.Errorf("Expected %d results, got %d", limit, len(results))
	}
	
	// Test search with filter
	filter := NewEqualsCondition("tag", "A")
	
	filteredResults, err := collection.Search(&QueryRequest{
		Vector: query,
		TopK:   10,
		Filter: filter,
		Params: &SearchParams{},
	})
	
	if err != nil {
		t.Errorf("Filtered search error: %v", err)
	}
	
	// Check filtered results (mock index should return vectors that match the filter)
	for _, res := range filteredResults {
		if res.Vector.Metadata["tag"] != "A" {
			t.Errorf("Filter failed: result %s has tag %v, expected A", 
				res.ID, res.Vector.Metadata["tag"])
		}
	}
	
	// Test search params
	params := &SearchParams{
		ScoreThreshold: 0.95,
	}
	
	_, err = collection.Search(&QueryRequest{
		Vector: query,
		TopK:   10,
		Filter: nil,
		Params: params,
	})
	
	if err != nil {
		t.Errorf("Search with params error: %v", err)
	}
}

func TestVectorCollectionBatchOperations(t *testing.T) {
	// Create collection with mock index
	collection, _ := NewVectorCollection("test", 3, Cosine)
	mockIndex := NewMockIndex(3, Cosine)
	collection.Index = mockIndex
	
	// Test batch insert
	vectors := []*Vector{
		NewVector("b1", []float32{1, 0, 0}, nil),
		NewVector("b2", []float32{0, 1, 0}, nil),
		NewVector("b3", []float32{0, 0, 1}, nil),
	}
	
	err := collection.BatchInsert(vectors)
	if err != nil {
		t.Errorf("Batch insert error: %v", err)
	}
	
	// Verify all vectors were inserted
	if size := collection.Size(); size != len(vectors) {
		t.Errorf("Expected size %d after batch insert, got %d", len(vectors), size)
	}
	
	for _, v := range vectors {
		if _, exists := collection.Get(v.ID); !exists {
			t.Errorf("Vector %s not found after batch insert", v.ID)
		}
	}
	
	// Test batch delete
	ids := []string{"b1", "b3"}
	
	err = collection.BatchDelete(ids)
	if err != nil {
		t.Errorf("Batch delete error: %v", err)
	}
	
	// Check size after batch delete
	expectedSize := len(vectors) - len(ids)
	if size := collection.Size(); size != expectedSize {
		t.Errorf("Expected size %d after batch delete, got %d", expectedSize, size)
	}
	
	// Verify deleted vectors are gone
	for _, id := range ids {
		if _, exists := collection.Get(id); exists {
			t.Errorf("Vector %s still exists after batch delete", id)
		}
	}
	
	// Check remaining vector
	if _, exists := collection.Get("b2"); !exists {
		t.Errorf("Vector b2 should still exist")
	}
}

func TestVectorCollectionSchema(t *testing.T) {
	// Create collection
	collection, _ := NewVectorCollection("test", 3, Cosine)
	
	// Initialize schema
	schema := NewMetadataSchema()
	schema.AddField("name", String, true)
	schema.AddField("age", Integer, false)
	schema.AddField("active", Boolean, true)
	
	// Set schema
	collection.SetSchema(schema)
	
	// Get schema and verify
	retrievedSchema := collection.GetSchema()
	if retrievedSchema == nil {
		t.Fatalf("Retrieved schema is nil")
	}
	
	// Basic schema verification (more detailed tests are in metadata_test.go)
	validMetadata := map[string]interface{}{
		"name":   "test",
		"active": true,
		"age":    30,
	}
	
	invalidMetadata := map[string]interface{}{
		"active": true,
		"age":    30,
		// Missing required "name" field
	}
	
	if err := retrievedSchema.Validate(validMetadata); err != nil {
		t.Errorf("Valid metadata failed schema validation: %v", err)
	}
	
	if err := retrievedSchema.Validate(invalidMetadata); err == nil {
		t.Errorf("Invalid metadata passed schema validation")
	}
	
	// Test insert with schema validation
	validVector := NewVector("valid", []float32{1, 0, 0}, validMetadata)
	invalidVector := NewVector("invalid", []float32{1, 0, 0}, invalidMetadata)
	
	// Replace index with mock to focus on schema validation
	mockIndex := NewMockIndex(3, Cosine)
	collection.Index = mockIndex
	
	// Valid insert should work
	if err := collection.Insert(validVector); err != nil {
		t.Errorf("Failed to insert valid vector: %v", err)
	}
	
	// Invalid insert should fail schema validation
	if err := collection.Insert(invalidVector); err == nil {
		t.Errorf("Expected schema validation error, got nil")
	}
}