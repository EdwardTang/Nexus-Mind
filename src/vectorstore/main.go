package vectorstore

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Config holds application configuration
type Config struct {
	// Node configuration
	NodeID        string
	HTTPAddr      string
	
	// Vector store configuration
	Dimensions    int
	DistanceFunc  string
	
	// Cluster configuration
	VirtualNodes      int
	ReplicationFactor int
	
	// Rebalancing configuration
	MaxConcurrentTransfers int
	BatchSize              int
	
	// Logging configuration
	LogLevel LogLevel
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		NodeID:                "node-1",
		HTTPAddr:              "127.0.0.1:8080",
		Dimensions:            1536,
		DistanceFunc:          "cosine",
		VirtualNodes:          10,
		ReplicationFactor:     3,
		MaxConcurrentTransfers: 3,
		BatchSize:             1000,
		LogLevel:              InfoLevel,
	}
}

// parseFlags parses command line flags into a Config
func parseFlags() Config {
	config := DefaultConfig()
	
	// Node configuration
	flag.StringVar(&config.NodeID, "node-id", config.NodeID, "Unique identifier for this node")
	flag.StringVar(&config.HTTPAddr, "http-addr", config.HTTPAddr, "HTTP server address")
	
	// Vector store configuration
	flag.IntVar(&config.Dimensions, "dimensions", config.Dimensions, "Vector dimensions")
	flag.StringVar(&config.DistanceFunc, "distance", config.DistanceFunc, "Distance function (cosine, dot, euclidean)")
	
	// Cluster configuration
	flag.IntVar(&config.VirtualNodes, "virtual-nodes", config.VirtualNodes, "Number of virtual nodes per physical node")
	flag.IntVar(&config.ReplicationFactor, "replication", config.ReplicationFactor, "Replication factor for vectors")
	
	// Rebalancing configuration
	flag.IntVar(&config.MaxConcurrentTransfers, "max-transfers", config.MaxConcurrentTransfers, "Maximum concurrent transfers")
	flag.IntVar(&config.BatchSize, "batch-size", config.BatchSize, "Batch size for transfers")
	
	// Logging configuration
	var logLevel int
	flag.IntVar(&logLevel, "log-level", int(config.LogLevel), "Log level (0=Debug, 1=Info, 2=Warn, 3=Error)")
	config.LogLevel = LogLevel(logLevel)
	
	flag.Parse()
	return config
}

func main() {
	// Parse configuration
	config := parseFlags()
	
	// Create logger
	logger := NewSimpleLogger(config.LogLevel, config.NodeID)
	logger.Info("Starting vector store node %s", config.NodeID)
	logger.Info("Configured for %d-dimensional vectors with %s distance", 
		config.Dimensions, config.DistanceFunc)
	
	// Initialize vector store
	vsConfig := VectorStoreConfig{
		NodeID:       config.NodeID,
		Dimensions:   config.Dimensions,
		DistanceFunc: config.DistanceFunc,
	}
	
	vectorStore, err := NewVectorStore(vsConfig, logger)
	if err != nil {
		logger.Error("Failed to create vector store: %v", err)
		os.Exit(1)
	}
	
	// Initialize token ring
	tokenRing := NewTokenRing(config.VirtualNodes, config.ReplicationFactor)
	
	// Add this node to the token ring
	tokenRing.AddNode(config.NodeID)
	vectorStore.SetTokenRing(tokenRing)
	
	// Initialize transfer service
	retryConfig := DefaultRetryConfig()
	transferLogger := NewSimpleLogger(config.LogLevel, fmt.Sprintf("%s-transfer", config.NodeID))
	transferService := NewTransferService(retryConfig, config.MaxConcurrentTransfers, transferLogger)
	transferService.SetVectorStore(vectorStore)
	
	// Initialize coordinator
	rebalanceConfig := DefaultRebalanceConfig()
	rebalanceConfig.MaxConcurrentTransfers = config.MaxConcurrentTransfers
	rebalanceConfig.BatchSize = config.BatchSize
	
	coordLogger := NewSimpleLogger(config.LogLevel, fmt.Sprintf("%s-coordinator", config.NodeID))
	coordinator := NewCoordinator(rebalanceConfig, coordLogger)
	coordinator.SetServices(nil, transferService, vectorStore, tokenRing)
	
	// Initialize HTTP server
	httpLogger := NewSimpleLogger(config.LogLevel, fmt.Sprintf("%s-http", config.NodeID))
	httpServer := NewHTTPServer(vectorStore, coordinator, httpLogger)
	
	// Create an HTTP server with the handlers
	server := &http.Server{
		Addr:    config.HTTPAddr,
		Handler: http.DefaultServeMux,
	}
	
	// Register routes
	http.HandleFunc("/vectors", httpServer.handleVectors)
	http.HandleFunc("/vectors/", httpServer.handleVectorByID)
	http.HandleFunc("/search", httpServer.handleSearch)
	http.HandleFunc("/stats", httpServer.handleStats)
	http.HandleFunc("/cluster", httpServer.handleCluster)
	
	// Start HTTP server in a goroutine
	go func() {
		logger.Info("HTTP server listening on %s", config.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed: %v", err)
			os.Exit(1)
		}
	}()
	
	logger.Info("Server started successfully. Press Ctrl+C to exit.")
	
	// Wait for interrupt signal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	
	logger.Info("Shutting down...")
	
	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error: %v", err)
	}
	
	logger.Info("Server stopped")
}