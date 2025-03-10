package query

import (
	"errors"
	"fmt"

	"course/models"
)

// Processor handles vector search queries with different strategies
type Processor struct {
	collection *models.VectorCollection
}

// NewProcessor creates a new query processor for a vector collection
func NewProcessor(collection *models.VectorCollection) *Processor {
	return &Processor{
		collection: collection,
	}
}

// ProcessQuery handles a unified query request, dispatching it to the appropriate handler
func (p *Processor) ProcessQuery(request *models.QueryRequest) (interface{}, error) {
	// Validate request
	if err := p.validateRequest(request); err != nil {
		return nil, err
	}

	// Initialize search parameters if not provided
	if request.Params == nil {
		request.Params = &models.SearchParams{
			SearchStrategy: models.Default,
			HnswEf:        100,
		}
	}

	// Determine which operation to perform based on query type
	switch {
	case request.Vector != nil:
		// Vector similarity search (kNN)
		return p.processVectorSearch(request)
	case request.PointID != "":
		// Search by existing point ID
		return p.processPointIDSearch(request)
	case request.Recommend != nil:
		// Recommendation by examples
		return p.processRecommendation(request)
	case request.Scroll != nil:
		// Pagination through all points
		return p.processScroll(request)
	case request.Sample != "":
		// Random sampling
		return p.processSample(request)
	default:
		return nil, errors.New("invalid query: no query type specified")
	}
}

// validateRequest checks if the query request is valid
func (p *Processor) validateRequest(request *models.QueryRequest) error {
	if request == nil {
		return errors.New("request cannot be nil")
	}

	// Check for valid limit
	if request.Limit <= 0 {
		request.Limit = 10 // Default limit
	}

	// Check that exactly one query type is specified
	queryTypes := 0
	if request.Vector != nil {
		queryTypes++
	}
	if request.PointID != "" {
		queryTypes++
	}
	if request.Recommend != nil {
		queryTypes++
	}
	if request.Scroll != nil {
		queryTypes++
	}
	if request.Sample != "" {
		queryTypes++
	}

	if queryTypes == 0 {
		return errors.New("no query type specified")
	}
	if queryTypes > 1 {
		return errors.New("multiple query types specified, only one is allowed")
	}

	// Validate specific query types
	if request.Vector != nil && len(request.Vector) != p.collection.Dimension {
		return fmt.Errorf("query vector dimension %d does not match collection dimension %d", 
			len(request.Vector), p.collection.Dimension)
	}

	if request.GroupBy != "" && (request.GroupSize <= 0 || request.GroupLimit <= 0) {
		request.GroupSize = 1  // Default group size
		request.GroupLimit = request.Limit // Default group limit
	}

	return nil
}

// processVectorSearch handles vector similarity search
func (p *Processor) processVectorSearch(request *models.QueryRequest) (interface{}, error) {
	// Adjust search parameters based on strategy
	p.adjustSearchParams(request.Params)

	// Perform the search
	results, err := p.collection.Search(
		request.Vector,
		request.Limit,
		request.Filter,
		request.Params,
	)
	if err != nil {
		return nil, err
	}

	// Handle grouping if requested
	if request.GroupBy != "" {
		return p.groupResults(results, request)
	}

	// Apply post-processing
	return p.postProcessResults(results, request)
}

// processPointIDSearch handles search by existing point ID
func (p *Processor) processPointIDSearch(request *models.QueryRequest) (interface{}, error) {
	// This is a stub implementation
	// In a real implementation, we would:
	// 1. Retrieve the vector with the given ID
	// 2. Use that vector as a query for a similarity search
	
	return nil, errors.New("search by point ID not implemented yet")
}

// processRecommendation handles recommendation by examples
func (p *Processor) processRecommendation(request *models.QueryRequest) (interface{}, error) {
	// This is a stub implementation
	// In a real implementation, we would:
	// 1. Retrieve the vectors for the positive and negative examples
	// 2. Combine them according to the recommendation strategy
	// 3. Use the combined vector as a query for a similarity search
	
	return nil, errors.New("recommendation search not implemented yet")
}

// processScroll handles pagination through all points
func (p *Processor) processScroll(request *models.QueryRequest) (interface{}, error) {
	// This is a stub implementation
	// In a real implementation, we would:
	// 1. Use the offset as a cursor to determine where to start
	// 2. Return a page of results and a new cursor
	
	return nil, errors.New("scroll not implemented yet")
}

// processSample handles random sampling
func (p *Processor) processSample(request *models.QueryRequest) (interface{}, error) {
	// This is a stub implementation
	// In a real implementation, we would:
	// 1. Randomly select 'limit' vectors from the collection
	// 2. Apply filters if provided
	
	return nil, errors.New("random sampling not implemented yet")
}

// adjustSearchParams modifies search parameters based on the search strategy
func (p *Processor) adjustSearchParams(params *models.SearchParams) {
	switch params.SearchStrategy {
	case models.ExactSearch:
		params.Exact = true
		params.HnswEf = 0 // Ignored in exact search
	case models.FastSearch:
		params.Exact = false
		if params.HnswEf == 0 {
			params.HnswEf = 40 // Lower ef for faster search
		}
	case models.PreciseSearch:
		params.Exact = false
		if params.HnswEf == 0 {
			params.HnswEf = 300 // Higher ef for more accurate search
		}
	case models.BatchSearch:
		// BatchSearch is handled differently, no special params
	default: // Default strategy
		params.Exact = false
		if params.HnswEf == 0 {
			params.HnswEf = 100 // Default ef value
		}
	}
}

// postProcessResults applies post-processing to search results
func (p *Processor) postProcessResults(results []models.SearchResult, request *models.QueryRequest) (interface{}, error) {
	// Apply offset if provided
	if request.Offset > 0 && request.Offset < len(results) {
		results = results[request.Offset:]
	}

	// Filter results by score threshold if provided
	if request.Params != nil && request.Params.ScoreThreshold > 0 {
		filteredResults := make([]models.SearchResult, 0, len(results))
		for _, result := range results {
			if result.Score >= request.Params.ScoreThreshold {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}

	// Remove vector data if not requested
	if !request.WithVectors {
		for i := range results {
			results[i].Vector = nil
		}
	}

	// For payload (metadata) control, we'd need to implement more complex logic
	// based on the value of request.WithPayload
	// This is a stub implementation:
	if !isWithPayload(request.WithPayload) {
		// Remove all metadata if not requested
		for i := range results {
			if results[i].Vector != nil {
				results[i].Vector.Metadata = nil
			}
		}
	}

	return results, nil
}

// groupResults groups search results by a metadata field
func (p *Processor) groupResults(results []models.SearchResult, request *models.QueryRequest) (interface{}, error) {
	// This is a stub implementation for grouping
	// In a real implementation, we would:
	// 1. Group results by the specified metadata field
	// 2. Apply group size and limit constraints
	// 3. Sort groups by the best result in each group
	
	return nil, errors.New("grouping not implemented yet")
}

// isWithPayload determines if payload (metadata) should be included in results
func isWithPayload(withPayload interface{}) bool {
	if withPayload == nil {
		return false
	}

	switch v := withPayload.(type) {
	case bool:
		return v
	case string:
		return true // Any string means include some payload
	case []string:
		return len(v) > 0 // Include if there are fields specified
	case map[string]interface{}:
		return true // Any map configuration means include some payload
	default:
		return false
	}
}