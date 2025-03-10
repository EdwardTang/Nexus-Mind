package query

import (
	"bytes"
	"course/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"fmt"
)

func TestAPICreation(t *testing.T) {
	// Create a processor
	processor := NewProcessor()
	
	// Create API with processor
	api := NewAPI(processor)
	
	if api == nil {
		t.Fatalf("Failed to create API")
	}
	
	if api.processor != processor {
		t.Errorf("API processor doesn't match the one provided")
	}
}

func TestSearchEndpoint(t *testing.T) {
	// Create processor with mock collection
	processor := NewProcessor()
	coll := NewMockCollection("test", 3)
	processor.RegisterCollection("test", coll)
	
	// Insert test vectors
	vectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0, 0}, map[string]interface{}{"category": "A"}),
		models.NewVector("v2", []float32{0, 1, 0}, map[string]interface{}{"category": "B"}),
	}
	
	for _, v := range vectors {
		coll.Insert(v)
	}
	
	// Create API
	api := NewAPI(processor)
	
	// Create search request
	searchReq := &SearchRequest{
		CollectionName: "test",
		QueryVector:    []float32{1, 0, 0},
		TopK:           1,
		Strategy:       "default",
	}
	
	// Convert to JSON
	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	// Create HTTP request
	req := httptest.NewRequest("POST", "/api/collections/test/search", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Create response recorder
	rec := httptest.NewRecorder()
	
	// Handle the request
	api.SearchHandler(rec, req)
	
	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	
	// Parse response
	var searchResp SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &searchResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	// Check results
	if len(searchResp.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(searchResp.Results))
	}
	
	// Test with invalid collection
	invalidReq := &SearchRequest{
		CollectionName: "nonexistent",
		QueryVector:    []float32{1, 0, 0},
		TopK:           1,
	}
	
	reqBody, _ = json.Marshal(invalidReq)
	req = httptest.NewRequest("POST", "/api/collections/nonexistent/search", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	
	api.SearchHandler(rec, req)
	
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid collection, got %d", rec.Code)
	}
	
	// Test with invalid request body
	req = httptest.NewRequest("POST", "/api/collections/test/search", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	
	api.SearchHandler(rec, req)
	
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestCollectionEndpoints(t *testing.T) {
	// Create processor
	processor := NewProcessor()
	
	// Create API
	api := NewAPI(processor)
	
	// Test create collection
	createReq := map[string]interface{}{
		"name":      "new_collection",
		"dimension": 4,
		"metric":    "cosine",
	}
	
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/collections", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	api.CreateCollectionHandler(rec, req)
	
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for collection creation, got %d", rec.Code)
	}
	
	// Verify collection was created
	if _, err := processor.GetCollection("new_collection"); err != nil {
		t.Errorf("Collection was not created properly: %v", err)
	}
	
	// Test list collections
	req = httptest.NewRequest("GET", "/api/collections", nil)
	rec = httptest.NewRecorder()
	
	api.ListCollectionsHandler(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for list collections, got %d", rec.Code)
	}
	
	var listResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &listResp)
	
	collections, ok := listResp["collections"].([]interface{})
	if !ok {
		t.Fatalf("Expected collections array in response")
	}
	
	if len(collections) != 1 {
		t.Errorf("Expected 1 collection, got %d", len(collections))
	}
	
	// Test delete collection
	req = httptest.NewRequest("DELETE", "/api/collections/new_collection", nil)
	rec = httptest.NewRecorder()
	
	api.DeleteCollectionHandler(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for delete collection, got %d", rec.Code)
	}
	
	// Verify collection was deleted
	if _, err := processor.GetCollection("new_collection"); err == nil {
		t.Errorf("Collection still exists after deletion")
	}
	
	// Test delete non-existent collection
	req = httptest.NewRequest("DELETE", "/api/collections/nonexistent", nil)
	rec = httptest.NewRecorder()
	
	api.DeleteCollectionHandler(rec, req)
	
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleting non-existent collection, got %d", rec.Code)
	}
}

func TestVectorEndpoints(t *testing.T) {
	// Create processor with collection
	processor := NewProcessor()
	coll := NewMockCollection("test", 3)
	processor.RegisterCollection("test", coll)
	
	// Create API
	api := NewAPI(processor)
	
	// Test insert vector
	vectorData := map[string]interface{}{
		"id":     "test_vector",
		"values": []float32{1, 2, 3},
		"metadata": map[string]interface{}{
			"key": "value",
		},
	}
	
	reqBody, _ := json.Marshal(vectorData)
	req := httptest.NewRequest("POST", "/api/collections/test/vectors", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	api.InsertVectorHandler(rec, req)
	
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for vector insertion, got %d", rec.Code)
	}
	
	// Test get vector
	req = httptest.NewRequest("GET", "/api/collections/test/vectors/test_vector", nil)
	rec = httptest.NewRecorder()
	
	api.GetVectorHandler(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for get vector, got %d", rec.Code)
	}
	
	var getResp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &getResp)
	
	if getResp["id"] != "test_vector" {
		t.Errorf("Expected vector ID test_vector, got %v", getResp["id"])
	}
	
	// Test delete vector
	req = httptest.NewRequest("DELETE", "/api/collections/test/vectors/test_vector", nil)
	rec = httptest.NewRecorder()
	
	api.DeleteVectorHandler(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for delete vector, got %d", rec.Code)
	}
	
	// Verify vector was deleted
	req = httptest.NewRequest("GET", "/api/collections/test/vectors/test_vector", nil)
	rec = httptest.NewRecorder()
	
	api.GetVectorHandler(rec, req)
	
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted vector, got %d", rec.Code)
	}
	
	// Test with non-existent collection
	req = httptest.NewRequest("POST", "/api/collections/nonexistent/vectors", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	
	api.InsertVectorHandler(rec, req)
	
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent collection, got %d", rec.Code)
	}
}

func TestBatchVectorOperations(t *testing.T) {
	// Create processor with collection
	processor := NewProcessor()
	coll := NewMockCollection("test", 3)
	processor.RegisterCollection("test", coll)
	
	// Create API
	api := NewAPI(processor)
	
	// Test batch insert
	batchData := map[string]interface{}{
		"vectors": []map[string]interface{}{
			{
				"id":     "batch1",
				"values": []float32{1, 0, 0},
			},
			{
				"id":     "batch2",
				"values": []float32{0, 1, 0},
			},
		},
	}
	
	reqBody, _ := json.Marshal(batchData)
	req := httptest.NewRequest("POST", "/api/collections/test/vectors/batch", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	
	api.BatchInsertVectorsHandler(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for batch insert, got %d", rec.Code)
	}
	
	// Test batch delete
	deleteData := map[string]interface{}{
		"ids": []string{"batch1", "batch2"},
	}
	
	reqBody, _ = json.Marshal(deleteData)
	req = httptest.NewRequest("POST", "/api/collections/test/vectors/batch/delete", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	
	api.BatchDeleteVectorsHandler(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for batch delete, got %d", rec.Code)
	}
	
	// Verify vectors were deleted
	if _, exists := coll.Get("batch1"); exists {
		t.Errorf("Vector batch1 still exists after batch delete")
	}
	
	if _, exists := coll.Get("batch2"); exists {
		t.Errorf("Vector batch2 still exists after batch delete")
	}
}