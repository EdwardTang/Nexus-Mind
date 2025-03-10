package models

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
)

func TestVectorCreation(t *testing.T) {
	// Test basic vector creation
	id := "test_vector"
	values := []float32{1.0, 2.0, 3.0, 4.0}
	metadata := map[string]interface{}{"key1": "value1", "key2": 42}

	vector := NewVector(id, values, metadata)

	if vector.ID != id {
		t.Errorf("Expected ID %s, got %s", id, vector.ID)
	}

	if !reflect.DeepEqual(vector.Values, values) {
		t.Errorf("Expected values %v, got %v", values, vector.Values)
	}

	if !reflect.DeepEqual(vector.Metadata, metadata) {
		t.Errorf("Expected metadata %v, got %v", metadata, vector.Metadata)
	}

	// Test dimension
	if vector.Dim() != len(values) {
		t.Errorf("Expected dimension %d, got %d", len(values), vector.Dim())
	}

	// Test nil metadata
	vector2 := NewVector("v2", values, nil)
	if vector2.Metadata == nil {
		t.Errorf("Expected non-nil metadata map, got nil")
	}
	if len(vector2.Metadata) != 0 {
		t.Errorf("Expected empty metadata map, got %v", vector2.Metadata)
	}
}

func TestVectorNormalize(t *testing.T) {
	// Create a vector
	values := []float32{3.0, 4.0} // Simple 3-4-5 triangle
	vector := NewVector("normalize_test", values, nil)

	// Get a normalized copy
	normalized := vector.Normalized()

	// Check original is unchanged
	if vector.Values[0] != 3.0 || vector.Values[1] != 4.0 {
		t.Errorf("Original vector was modified: %v", vector.Values)
	}

	// Expected normalized values (3/5, 4/5)
	expected := []float32{0.6, 0.8}
	epsilon := float32(0.0001)

	// Check normalized values
	for i, v := range normalized.Values {
		if abs(v-expected[i]) > epsilon {
			t.Errorf("Expected normalized[%d] = %f, got %f", i, expected[i], v)
		}
	}

	// Check in-place normalization
	vector.Normalize()
	for i, v := range vector.Values {
		if abs(v-expected[i]) > epsilon {
			t.Errorf("Expected in-place normalized[%d] = %f, got %f", i, expected[i], v)
		}
	}
}

func TestVectorCopy(t *testing.T) {
	// Create original vector
	id := "original"
	values := []float32{1.0, 2.0, 3.0}
	metadata := map[string]interface{}{"key": "value"}
	original := NewVector(id, values, metadata)

	// Make a copy
	copy := original.Copy()

	// Check they're equal but not the same instance
	if copy == original {
		t.Errorf("Copy returned the same instance")
	}

	if copy.ID != original.ID {
		t.Errorf("Expected copy ID %s, got %s", original.ID, copy.ID)
	}

	// Check values are equal but not the same array
	if !reflect.DeepEqual(copy.Values, original.Values) {
		t.Errorf("Expected copy values %v, got %v", original.Values, copy.Values)
	}
	
	// Ensure the underlying arrays are different (changing one shouldn't affect the other)
	original.Values[0] = 999.0
	if copy.Values[0] == 999.0 {
		t.Errorf("Modifying original affected the copy")
	}

	// Check metadata is equal but not the same map
	if !reflect.DeepEqual(copy.Metadata, original.Metadata) {
		t.Errorf("Expected copy metadata %v, got %v", original.Metadata, copy.Metadata)
	}
	
	// Modify original metadata
	original.Metadata["new_key"] = "new_value"
	if _, exists := copy.Metadata["new_key"]; exists {
		t.Errorf("Modifying original metadata affected the copy")
	}
}

func TestVectorSerialization(t *testing.T) {
	// Create a vector
	id := "serialize_test"
	values := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	metadata := map[string]interface{}{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"nested": map[string]interface{}{
			"inner": "nested value",
		},
	}
	vector := NewVector(id, values, metadata)

	// Test serializability
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(vector)
	if err != nil {
		t.Fatalf("Failed to encode vector: %v", err)
	}

	// Decode and verify
	dec := gob.NewDecoder(&buf)
	var decoded Vector
	err = dec.Decode(&decoded)
	if err != nil {
		t.Fatalf("Failed to decode vector: %v", err)
	}

	// Check everything matches
	if decoded.ID != vector.ID {
		t.Errorf("Expected ID %s, got %s", vector.ID, decoded.ID)
	}

	if !reflect.DeepEqual(decoded.Values, vector.Values) {
		t.Errorf("Expected values %v, got %v", vector.Values, decoded.Values)
	}

	// Basic check for top-level metadata keys
	for k, v := range vector.Metadata {
		if !reflect.DeepEqual(decoded.Metadata[k], v) && k != "nested" {
			t.Errorf("Metadata mismatch for key %s: expected %v, got %v", 
				k, v, decoded.Metadata[k])
		}
	}
}

func TestSparseVector(t *testing.T) {
	// Create a sparse vector
	id := "sparse_test"
	indices := []int{0, 2, 5, 9}
	values := []float32{1.0, 2.0, 3.0, 4.0}
	dim := 10
	metadata := map[string]interface{}{"key": "value"}

	sparse := NewSparseVector(id, indices, values, dim, metadata)

	// Check properties
	if sparse.ID != id {
		t.Errorf("Expected ID %s, got %s", id, sparse.ID)
	}

	if !reflect.DeepEqual(sparse.Indices, indices) {
		t.Errorf("Expected indices %v, got %v", indices, sparse.Indices)
	}

	if !reflect.DeepEqual(sparse.Values, values) {
		t.Errorf("Expected values %v, got %v", values, sparse.Values)
	}

	if sparse.Dimension != dim {
		t.Errorf("Expected dimension %d, got %d", dim, sparse.Dimension)
	}

	// Test to dense conversion
	dense := sparse.ToDense()

	// Check the dense vector
	expectedDense := []float32{1.0, 0.0, 2.0, 0.0, 0.0, 3.0, 0.0, 0.0, 0.0, 4.0}
	if !reflect.DeepEqual(dense.Values, expectedDense) {
		t.Errorf("Expected dense values %v, got %v", expectedDense, dense.Values)
	}

	// Check metadata was transferred
	if !reflect.DeepEqual(dense.Metadata, metadata) {
		t.Errorf("Expected metadata %v, got %v", metadata, dense.Metadata)
	}

	// Test dimension
	if dense.Dim() != dim {
		t.Errorf("Expected dimension %d, got %d", dim, dense.Dim())
	}
}

func TestSparseSerialization(t *testing.T) {
	// Create a sparse vector
	id := "sparse_serialize_test"
	indices := []int{1, 3, 5}
	values := []float32{10.0, 20.0, 30.0}
	dim := 10
	metadata := map[string]interface{}{
		"key": "value",
	}
	sparse := NewSparseVector(id, indices, values, dim, metadata)

	// Test serializability
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(sparse)
	if err != nil {
		t.Fatalf("Failed to encode sparse vector: %v", err)
	}

	// Decode and verify
	dec := gob.NewDecoder(&buf)
	var decoded SparseVector
	err = dec.Decode(&decoded)
	if err != nil {
		t.Fatalf("Failed to decode sparse vector: %v", err)
	}

	// Check everything matches
	if decoded.ID != sparse.ID {
		t.Errorf("Expected ID %s, got %s", sparse.ID, decoded.ID)
	}

	if !reflect.DeepEqual(decoded.Indices, sparse.Indices) {
		t.Errorf("Expected indices %v, got %v", sparse.Indices, decoded.Indices)
	}

	if !reflect.DeepEqual(decoded.Values, sparse.Values) {
		t.Errorf("Expected values %v, got %v", sparse.Values, decoded.Values)
	}

	if decoded.Dimension != sparse.Dimension {
		t.Errorf("Expected dimension %d, got %d", sparse.Dimension, decoded.Dimension)
	}

	// Check metadata
	if !reflect.DeepEqual(decoded.Metadata, sparse.Metadata) {
		t.Errorf("Expected metadata %v, got %v", sparse.Metadata, decoded.Metadata)
	}
}

// Helper function for float comparison
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}