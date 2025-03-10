package query

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"course/models"
)

// API provides a RESTful interface to the vector store
type API struct {
	collections map[string]*models.VectorCollection
	processors  map[string]*Processor
}

// NewAPI creates a new API instance
func NewAPI() *API {
	return &API{
		collections: make(map[string]*models.VectorCollection),
		processors:  make(map[string]*Processor),
	}
}

// RegisterCollection adds a collection to the API
func (api *API) RegisterCollection(collection *models.VectorCollection) {
	api.collections[collection.Name] = collection
	api.processors[collection.Name] = NewProcessor(collection)
}

// SetupRoutes configures HTTP routes for the API
func (api *API) SetupRoutes(mux *http.ServeMux) {
	// Collection management
	mux.HandleFunc("/collections", api.handleCollections)
	mux.HandleFunc("/collections/", api.handleCollectionOperations)
}

// handleCollections handles requests to /collections
func (api *API) handleCollections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List all collections
		api.listCollections(w, r)
	case http.MethodPost:
		// Create a new collection
		api.createCollection(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleCollectionOperations handles requests to /collections/{name}/...
func (api *API) handleCollectionOperations(w http.ResponseWriter, r *http.Request) {
	// Extract collection name from path
	path := strings.TrimPrefix(r.URL.Path, "/collections/")
	parts := strings.SplitN(path, "/", 2)
	
	if len(parts) == 0 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	collectionName := parts[0]
	collection, exists := api.collections[collectionName]
	if !exists {
		http.Error(w, fmt.Sprintf("Collection %s not found", collectionName), http.StatusNotFound)
		return
	}
	
	// Handle operations on the collection
	if len(parts) == 1 {
		// Operations on the collection itself
		switch r.Method {
		case http.MethodGet:
			// Get collection info
			api.getCollection(w, r, collectionName)
		case http.MethodDelete:
			// Delete collection
			api.deleteCollection(w, r, collectionName)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// Operations on collection resources
	resource := parts[1]
	
	// Vector operations
	if strings.HasPrefix(resource, "vectors") {
		api.handleVectorOperations(w, r, collection, strings.TrimPrefix(resource, "vectors"))
		return
	}
	
	// Query operations
	if strings.HasPrefix(resource, "query") {
		api.handleQueryOperations(w, r, collectionName, strings.TrimPrefix(resource, "query"))
		return
	}
	
	http.Error(w, "Resource not found", http.StatusNotFound)
}

// listCollections returns a list of all collections
func (api *API) listCollections(w http.ResponseWriter, r *http.Request) {
	collections := make([]map[string]interface{}, 0, len(api.collections))
	
	for name, coll := range api.collections {
		collections = append(collections, map[string]interface{}{
			"name":      name,
			"dimension": coll.Dimension,
			"metric":    coll.DistanceFunc.String(),
			"vectors":   coll.Size(),
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"collections": collections,
		"status":      "ok",
	})
}

// createCollection creates a new vector collection
func (api *API) createCollection(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name      string `json:"name"`
		Dimension int    `json:"dimension"`
		Metric    string `json:"metric"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if request.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	
	if request.Dimension <= 0 {
		http.Error(w, "Dimension must be positive", http.StatusBadRequest)
		return
	}
	
	// Check if collection already exists
	if _, exists := api.collections[request.Name]; exists {
		http.Error(w, fmt.Sprintf("Collection %s already exists", request.Name), http.StatusConflict)
		return
	}
	
	// Parse metric
	var metric models.DistanceMetric
	switch strings.ToLower(request.Metric) {
	case "cosine":
		metric = models.Cosine
	case "dotproduct", "dot_product", "dot":
		metric = models.DotProduct
	case "euclidean", "euclid", "l2":
		metric = models.Euclidean
	case "manhattan", "taxicab", "cityblock", "l1":
		metric = models.Manhattan
	default:
		metric = models.Cosine // Default to cosine
	}
	
	// Create collection
	collection := models.NewVectorCollection(request.Name, request.Dimension, metric)
	api.RegisterCollection(collection)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":      collection.Name,
		"dimension": collection.Dimension,
		"metric":    collection.DistanceFunc.String(),
		"status":    "created",
	})
}

// getCollection returns information about a collection
func (api *API) getCollection(w http.ResponseWriter, r *http.Request, name string) {
	collection, exists := api.collections[name]
	if !exists {
		http.Error(w, fmt.Sprintf("Collection %s not found", name), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":      collection.Name,
		"dimension": collection.Dimension,
		"metric":    collection.DistanceFunc.String(),
		"vectors":   collection.Size(),
		"status":    "ok",
	})
}

// deleteCollection removes a collection
func (api *API) deleteCollection(w http.ResponseWriter, r *http.Request, name string) {
	// Check if collection exists
	if _, exists := api.collections[name]; !exists {
		http.Error(w, fmt.Sprintf("Collection %s not found", name), http.StatusNotFound)
		return
	}
	
	// Delete collection
	delete(api.collections, name)
	delete(api.processors, name)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "deleted",
	})
}

// handleVectorOperations handles operations on vectors
func (api *API) handleVectorOperations(w http.ResponseWriter, r *http.Request, collection *models.VectorCollection, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	// Handle batch operations
	if len(parts) == 1 && parts[0] == "batch" {
		switch r.Method {
		case http.MethodPost:
			api.batchInsertVectors(w, r, collection)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// Handle operations on a specific vector
	if len(parts) == 1 && parts[0] != "" {
		vectorID := parts[0]
		switch r.Method {
		case http.MethodGet:
			api.getVector(w, r, collection, vectorID)
		case http.MethodDelete:
			api.deleteVector(w, r, collection, vectorID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
	
	// Handle operations on all vectors
	switch r.Method {
	case http.MethodGet:
		// List vectors (with pagination)
		api.listVectors(w, r, collection)
	case http.MethodPost, http.MethodPut:
		// Add or update a vector
		api.upsertVector(w, r, collection)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleQueryOperations handles query operations
func (api *API) handleQueryOperations(w http.ResponseWriter, r *http.Request, collectionName, path string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	processor, exists := api.processors[collectionName]
	if !exists {
		http.Error(w, fmt.Sprintf("Collection %s not found", collectionName), http.StatusNotFound)
		return
	}
	
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	// Handle batch query
	if len(parts) == 1 && parts[0] == "batch" {
		api.batchQuery(w, r, processor)
		return
	}
	
	// Handle groups query
	if len(parts) == 1 && parts[0] == "groups" {
		api.groupsQuery(w, r, processor)
		return
	}
	
	// Handle regular query
	api.query(w, r, processor)
}

// query handles a regular vector query
func (api *API) query(w http.ResponseWriter, r *http.Request, processor *Processor) {
	var request models.QueryRequest
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Process the query
	results, err := processor.ProcessQuery(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Return the results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": results,
		"status": "ok",
	})
}

// batchQuery handles batch queries
func (api *API) batchQuery(w http.ResponseWriter, r *http.Request, processor *Processor) {
	var request struct {
		Searches []models.QueryRequest `json:"searches"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Process each query
	results := make([]interface{}, len(request.Searches))
	for i, search := range request.Searches {
		result, err := processor.ProcessQuery(&search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results[i] = result
	}
	
	// Return the results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": results,
		"status": "ok",
	})
}

// groupsQuery handles queries with grouping
func (api *API) groupsQuery(w http.ResponseWriter, r *http.Request, processor *Processor) {
	var request models.QueryRequest
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Ensure GroupBy is set
	if request.GroupBy == "" {
		http.Error(w, "GroupBy is required for group queries", http.StatusBadRequest)
		return
	}
	
	// Process the query
	results, err := processor.ProcessQuery(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Return the results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": results,
		"status": "ok",
	})
}

// The following methods are stubs for vector operations - they would need to be implemented
// in a real application

func (api *API) upsertVector(w http.ResponseWriter, r *http.Request, collection *models.VectorCollection) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Not implemented",
		"status":  "error",
	})
}

func (api *API) batchInsertVectors(w http.ResponseWriter, r *http.Request, collection *models.VectorCollection) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Not implemented",
		"status":  "error",
	})
}

func (api *API) getVector(w http.ResponseWriter, r *http.Request, collection *models.VectorCollection, id string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Not implemented",
		"status":  "error",
	})
}

func (api *API) deleteVector(w http.ResponseWriter, r *http.Request, collection *models.VectorCollection, id string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Not implemented",
		"status":  "error",
	})
}

func (api *API) listVectors(w http.ResponseWriter, r *http.Request, collection *models.VectorCollection) {
	// Get pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 10
	offset := 0
	
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
	}
	
	if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			http.Error(w, "Invalid offset parameter", http.StatusBadRequest)
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Not implemented",
		"status":  "error",
	})
}