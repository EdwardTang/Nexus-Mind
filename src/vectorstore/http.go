package vectorstore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// HTTPServer provides a simple REST API for the vector store
type HTTPServer struct {
	vectorStore *VectorStore
	coordinator *Coordinator
	logger      Logger
}

// NewHTTPServer creates a new HTTP server for the vector store
func NewHTTPServer(vs *VectorStore, coord *Coordinator, logger Logger) *HTTPServer {
	return &HTTPServer{
		vectorStore: vs,
		coordinator: coord,
		logger:      logger,
	}
}

// Start starts the HTTP server on the specified address
func (s *HTTPServer) Start(addr string) error {
	// Register API routes
	http.HandleFunc("/vectors", s.handleVectors)
	http.HandleFunc("/vectors/", s.handleVectorByID)
	http.HandleFunc("/search", s.handleSearch)
	http.HandleFunc("/stats", s.handleStats)
	http.HandleFunc("/cluster", s.handleCluster)

	s.logger.Info("Starting HTTP server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleVectors handles POST requests to add vectors
func (s *HTTPServer) handleVectors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.logger.Debug("Rejected request with method %s to /vectors", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var vector Vector
	if err := json.NewDecoder(r.Body).Decode(&vector); err != nil {
		s.logger.Error("Failed to parse vector request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.vectorStore.AddVector(&vector); err != nil {
		s.logger.Error("Failed to add vector: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Added vector with ID: %s", vector.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": vector.ID})
}

// handleVectorByID handles GET and DELETE requests for a specific vector
func (s *HTTPServer) handleVectorByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/vectors/")
	if id == "" {
		s.logger.Debug("Rejected request with empty vector ID")
		http.Error(w, "Vector ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		vector, err := s.vectorStore.GetVector(id)
		if err != nil {
			s.logger.Debug("Vector not found: %s, error: %v", id, err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.logger.Debug("Retrieved vector with ID: %s", id)
		json.NewEncoder(w).Encode(vector)

	case http.MethodDelete:
		if err := s.vectorStore.DeleteVector(id); err != nil {
			s.logger.Debug("Failed to delete vector %s: %v", id, err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.logger.Debug("Deleted vector with ID: %s", id)
		w.WriteHeader(http.StatusNoContent)

	default:
		s.logger.Debug("Rejected %s request to /vectors/%s", r.Method, id)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SearchRequest represents a vector search request
type SearchRequest struct {
	Query    []float32           `json:"query"`
	K        int                 `json:"k"`
	Metadata map[string]string   `json:"metadata,omitempty"`
}

// handleSearch handles POST requests for vector similarity search
func (s *HTTPServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.logger.Debug("Rejected %s request to /search", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Failed to parse search request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a filter function if metadata filters are provided
	var filter func(*Vector) bool
	if len(req.Metadata) > 0 {
		filter = func(v *Vector) bool {
			for key, value := range req.Metadata {
				if metaVal, ok := v.Metadata[key].(string); !ok || metaVal != value {
					return false
				}
			}
			return true
		}
	}

	s.logger.Debug("Searching for %d nearest neighbors with %d filters", req.K, len(req.Metadata))
	results, err := s.vectorStore.Search(req.Query, req.K, filter)
	if err != nil {
		s.logger.Error("Search failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.logger.Debug("Search returned %d results", len(results))
	json.NewEncoder(w).Encode(results)
}

// handleStats handles GET requests for system statistics
func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.logger.Debug("Rejected %s request to /stats", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.vectorStore.GetStats()
	
	// Add coordinator metrics if available
	if s.coordinator != nil {
		metrics := s.coordinator.GetCurrentMetrics()
		if metrics != nil {
			stats["rebalance"] = metrics
		}
	}

	s.logger.Debug("Returning stats with %d vectors", stats["totalVectors"])
	json.NewEncoder(w).Encode(stats)
}

// handleCluster handles GET and POST requests for cluster operations
func (s *HTTPServer) handleCluster(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return cluster information
		operations := s.coordinator.GetAllOperations()
		info := map[string]interface{}{
			"operations": operations,
		}
		s.logger.Debug("Returning cluster info with %d operations", len(operations))
		json.NewEncoder(w).Encode(info)

	case http.MethodPost:
		// Trigger a rebalancing operation
		var req struct {
			Action string `json:"action"`
			NodeID string `json:"nodeId,omitempty"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.logger.Error("Failed to parse cluster request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		switch req.Action {
		case "join":
			if req.NodeID == "" {
				s.logger.Warn("Join request missing nodeId")
				http.Error(w, "NodeID required for join action", http.StatusBadRequest)
				return
			}
			// Simulate a node joining the cluster
			event := ClusterChangeEvent{
				Type:      NodeJoined,
				NodeID:    req.NodeID,
				Timestamp: 0, // Will be set by the membership service
			}
			s.logger.Info("Simulating node join: %s", req.NodeID)
			operationID := s.coordinator.TriggerRebalancing([]ClusterChangeEvent{event})
			json.NewEncoder(w).Encode(map[string]string{"operationId": operationID})

		case "leave":
			if req.NodeID == "" {
				s.logger.Warn("Leave request missing nodeId")
				http.Error(w, "NodeID required for leave action", http.StatusBadRequest)
				return
			}
			// Simulate a node leaving the cluster
			event := ClusterChangeEvent{
				Type:      NodeLeft,
				NodeID:    req.NodeID,
				Timestamp: 0, // Will be set by the membership service
			}
			s.logger.Info("Simulating node leave: %s", req.NodeID)
			operationID := s.coordinator.TriggerRebalancing([]ClusterChangeEvent{event})
			json.NewEncoder(w).Encode(map[string]string{"operationId": operationID})

		default:
			s.logger.Warn("Unknown cluster action requested: %s", req.Action)
			http.Error(w, fmt.Sprintf("Unknown action: %s", req.Action), http.StatusBadRequest)
		}

	default:
		s.logger.Debug("Rejected %s request to /cluster", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}