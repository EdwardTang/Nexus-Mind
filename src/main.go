package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"course/models"
	"course/vector"
	"course/vector/index"
	"course/vector/query"
)

func main() {
	fmt.Println("Starting Nexus-Mind Vector Store...")

	// Create a sample collection with a linear index
	collection := createSampleCollection()

	// Set up the HTTP API
	api := query.NewAPI()
	api.RegisterCollection(collection)

	// Configure HTTP routes
	mux := http.NewServeMux()
	api.SetupRoutes(mux)

	// Start the HTTP server
	port := "8080"
	fmt.Printf("Starting HTTP server on port %s...\n", port)
	
	// Handle signals for graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := http.ListenAndServe(":"+port, mux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	fmt.Println("Server is running. Press Ctrl+C to stop.")

	// Wait for interrupt signal
	<-done
	fmt.Println("\nShutting down server...")
}

// createSampleCollection creates a sample vector collection with some test data
func createSampleCollection() *models.VectorCollection {
	// Create a collection
	collection := models.NewVectorCollection("sample", 3, models.Cosine)

	// Create a linear index
	linearIndex, err := index.NewLinearIndex(3, models.Cosine)
	if err != nil {
		log.Fatalf("Failed to create linear index: %v", err)
	}

	// Add the index to the collection
	if err := collection.AddIndex("linear", linearIndex); err != nil {
		log.Fatalf("Failed to add index to collection: %v", err)
	}

	// Add some sample vectors
	sampleVectors := []*models.Vector{
		models.NewVector("v1", []float32{1, 0, 0}, map[string]interface{}{
			"category": "electronics",
			"price":    299.99,
			"brand":    "Apple",
		}),
		models.NewVector("v2", []float32{0, 1, 0}, map[string]interface{}{
			"category": "electronics",
			"price":    199.99,
			"brand":    "Samsung",
		}),
		models.NewVector("v3", []float32{0, 0, 1}, map[string]interface{}{
			"category": "clothing",
			"price":    49.99,
			"brand":    "Nike",
		}),
		models.NewVector("v4", []float32{0.5, 0.5, 0}, map[string]interface{}{
			"category": "electronics",
			"price":    149.99,
			"brand":    "Google",
		}),
		models.NewVector("v5", []float32{0.8, 0.1, 0.1}, map[string]interface{}{
			"category": "electronics",
			"price":    349.99,
			"brand":    "Apple",
		}),
		models.NewVector("v6", []float32{0.1, 0.8, 0.1}, map[string]interface{}{
			"category": "electronics",
			"price":    249.99,
			"brand":    "Samsung",
		}),
		models.NewVector("v7", []float32{0.1, 0.1, 0.8}, map[string]interface{}{
			"category": "clothing",
			"price":    79.99,
			"brand":    "Adidas",
		}),
		models.NewVector("v8", []float32{0.33, 0.33, 0.33}, map[string]interface{}{
			"category": "home",
			"price":    129.99,
			"brand":    "IKEA",
		}),
	}

	// Insert vectors
	for _, v := range sampleVectors {
		if err := collection.Insert(v); err != nil {
			log.Printf("Failed to insert vector %s: %v", v.ID, err)
		}
	}

	fmt.Printf("Created sample collection with %d vectors\n", collection.Size())

	// Print example search query to show how to use the API
	fmt.Println("\nAPI Usage Examples:")
	fmt.Println("\n1. List collections:")
	fmt.Println("   GET /collections")
	
	fmt.Println("\n2. Search vectors:")
	fmt.Println("   POST /collections/sample/query")
	fmt.Println("   {")
	fmt.Println("     \"vector\": [1, 0, 0],")
	fmt.Println("     \"limit\": 5,")
	fmt.Println("     \"filter\": {")
	fmt.Println("       \"conditions\": [")
	fmt.Println("         {")
	fmt.Println("           \"field\": \"category\",")
	fmt.Println("           \"operator\": \"eq\",")
	fmt.Println("           \"value\": \"electronics\"")
	fmt.Println("         }")
	fmt.Println("       ],")
	fmt.Println("       \"operator\": 0")
	fmt.Println("     },")
	fmt.Println("     \"with_vectors\": true,")
	fmt.Println("     \"with_payload\": true")
	fmt.Println("   }")

	return collection
}