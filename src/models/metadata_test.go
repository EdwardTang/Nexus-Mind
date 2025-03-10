package models

import (
	"reflect"
	"testing"
)

func TestMetadataSchema(t *testing.T) {
	// Create a schema
	schema := NewMetadataSchema()
	
	// Add field definitions
	schema.AddField("name", String, true)
	schema.AddField("age", Integer, false)
	schema.AddField("active", Boolean, true)
	schema.AddField("score", Float, false)
	schema.AddField("tags", Array, false)
	
	// Check field validation
	validMetadata := map[string]interface{}{
		"name":   "test user",
		"age":    30,
		"active": true,
		"score":  92.5,
		"tags":   []string{"tag1", "tag2"},
	}
	
	invalidMetadata1 := map[string]interface{}{
		"age":    "thirty", // Wrong type (string instead of integer)
		"active": true,
		"name":   "test user",
	}
	
	invalidMetadata2 := map[string]interface{}{
		"age":    30,
		"active": true,
		// Missing required "name" field
	}
	
	// Test validation
	if err := schema.Validate(validMetadata); err != nil {
		t.Errorf("Valid metadata failed validation: %v", err)
	}
	
	if err := schema.Validate(invalidMetadata1); err == nil {
		t.Errorf("Invalid metadata type passed validation")
	}
	
	if err := schema.Validate(invalidMetadata2); err == nil {
		t.Errorf("Metadata missing required field passed validation")
	}
	
	// Test JSON schema generation
	jsonSchema := schema.JSONSchema()
	
	// Basic checks on the generated JSON schema
	if jsonSchema["type"] != "object" {
		t.Errorf("Expected schema type 'object', got %v", jsonSchema["type"])
	}
	
	properties, ok := jsonSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected properties to be a map, got %T", jsonSchema["properties"])
	}
	
	// Check name field
	nameField, ok := properties["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected name field to be a map")
	}
	if nameField["type"] != "string" {
		t.Errorf("Expected name field type to be string, got %v", nameField["type"])
	}
	
	// Check required fields
	required, ok := jsonSchema["required"].([]string)
	if !ok {
		t.Fatalf("Expected required to be a string array")
	}
	
	// name and active should be required
	foundName := false
	foundActive := false
	for _, field := range required {
		if field == "name" {
			foundName = true
		}
		if field == "active" {
			foundActive = true
		}
	}
	
	if !foundName {
		t.Errorf("Expected 'name' to be in required fields")
	}
	if !foundActive {
		t.Errorf("Expected 'active' to be in required fields")
	}
}

func TestFilterCondition(t *testing.T) {
	// Test equals condition
	eqCond := NewEqualsCondition("category", "books")
	
	// Should match
	if !eqCond.Matches(map[string]interface{}{"category": "books"}) {
		t.Errorf("Equals condition should match exact value")
	}
	
	// Shouldn't match
	if eqCond.Matches(map[string]interface{}{"category": "movies"}) {
		t.Errorf("Equals condition shouldn't match different value")
	}
	if eqCond.Matches(map[string]interface{}{"different_field": "books"}) {
		t.Errorf("Equals condition shouldn't match different field")
	}
	
	// Test range condition
	rangeCond := NewRangeCondition("age", 18, 30)
	
	// Should match
	if !rangeCond.Matches(map[string]interface{}{"age": 20}) {
		t.Errorf("Range condition should match value in range")
	}
	if !rangeCond.Matches(map[string]interface{}{"age": 18}) {
		t.Errorf("Range condition should match lower bound")
	}
	if !rangeCond.Matches(map[string]interface{}{"age": 30}) {
		t.Errorf("Range condition should match upper bound")
	}
	
	// Shouldn't match
	if rangeCond.Matches(map[string]interface{}{"age": 17}) {
		t.Errorf("Range condition shouldn't match value below range")
	}
	if rangeCond.Matches(map[string]interface{}{"age": 31}) {
		t.Errorf("Range condition shouldn't match value above range")
	}
	if rangeCond.Matches(map[string]interface{}{"age": "twenty"}) {
		t.Errorf("Range condition shouldn't match non-numeric value")
	}
	
	// Test contains condition for array
	containsCond := NewContainsCondition("tags", "important")
	
	// Should match
	if !containsCond.Matches(map[string]interface{}{"tags": []string{"important", "featured"}}) {
		t.Errorf("Contains condition should match array containing value")
	}
	
	// Shouldn't match
	if containsCond.Matches(map[string]interface{}{"tags": []string{"normal", "regular"}}) {
		t.Errorf("Contains condition shouldn't match array not containing value")
	}
	if containsCond.Matches(map[string]interface{}{"tags": "important"}) {
		t.Errorf("Contains condition shouldn't match non-array value")
	}
}

func TestNestedFieldAccess(t *testing.T) {
	// Test metadata with nested fields
	metadata := map[string]interface{}{
		"user": map[string]interface{}{
			"profile": map[string]interface{}{
				"name": "John Doe",
				"age":  30,
			},
			"settings": map[string]interface{}{
				"notifications": true,
			},
		},
		"tags": []string{"important", "featured"},
	}
	
	// Create a condition on a nested field
	nameCond := NewEqualsCondition("user.profile.name", "John Doe")
	ageCond := NewRangeCondition("user.profile.age", 25, 35)
	notifCond := NewEqualsCondition("user.settings.notifications", true)
	
	// Test nested field conditions
	if !nameCond.Matches(metadata) {
		t.Errorf("Nested field condition should match valid nested value")
	}
	
	if !ageCond.Matches(metadata) {
		t.Errorf("Nested range condition should match valid nested value")
	}
	
	if !notifCond.Matches(metadata) {
		t.Errorf("Nested boolean condition should match valid nested value")
	}
	
	// Test with invalid path
	invalidCond := NewEqualsCondition("user.profile.email", "john@example.com")
	if invalidCond.Matches(metadata) {
		t.Errorf("Condition with nonexistent nested field shouldn't match")
	}
}

func TestMetadataFilter(t *testing.T) {
	// Create test metadata
	metadata := map[string]interface{}{
		"category": "electronics",
		"price":    500,
		"inStock":  true,
		"tags":     []string{"sale", "featured", "new"},
		"specs": map[string]interface{}{
			"weight": 1.5,
			"color":  "black",
		},
	}
	
	// Test AND filter
	andFilter := NewAndFilter(
		NewEqualsCondition("category", "electronics"),
		NewRangeCondition("price", 100, 1000),
	)
	
	if !andFilter.Matches(metadata) {
		t.Errorf("AND filter should match when all conditions match")
	}
	
	// Add a non-matching condition
	andFilterWithFalse := NewAndFilter(
		NewEqualsCondition("category", "electronics"),
		NewEqualsCondition("inStock", false),
	)
	
	if andFilterWithFalse.Matches(metadata) {
		t.Errorf("AND filter shouldn't match when any condition doesn't match")
	}
	
	// Test OR filter
	orFilter := NewOrFilter(
		NewEqualsCondition("category", "clothing"), // Doesn't match
		NewRangeCondition("price", 400, 600),       // Matches
	)
	
	if !orFilter.Matches(metadata) {
		t.Errorf("OR filter should match when any condition matches")
	}
	
	// All conditions don't match
	orFilterAllFalse := NewOrFilter(
		NewEqualsCondition("category", "clothing"),
		NewRangeCondition("price", 1000, 2000),
	)
	
	if orFilterAllFalse.Matches(metadata) {
		t.Errorf("OR filter shouldn't match when all conditions don't match")
	}
	
	// Test NOT filter
	notFilter := NewNotFilter(
		NewEqualsCondition("category", "clothing"),
	)
	
	if !notFilter.Matches(metadata) {
		t.Errorf("NOT filter should match when inner condition doesn't match")
	}
	
	notFilterWithMatch := NewNotFilter(
		NewEqualsCondition("category", "electronics"),
	)
	
	if notFilterWithMatch.Matches(metadata) {
		t.Errorf("NOT filter shouldn't match when inner condition matches")
	}
	
	// Test complex nested filter
	complexFilter := NewAndFilter(
		NewOrFilter(
			NewEqualsCondition("category", "electronics"),
			NewEqualsCondition("category", "computers"),
		),
		NewRangeCondition("price", 200, 800),
		NewContainsCondition("tags", "sale"),
		NewEqualsCondition("specs.color", "black"),
	)
	
	if !complexFilter.Matches(metadata) {
		t.Errorf("Complex filter should match valid metadata")
	}
}

func TestFilterJSON(t *testing.T) {
	// Create a complex filter
	filter := NewAndFilter(
		NewEqualsCondition("category", "electronics"),
		NewRangeCondition("price", 100, 1000),
		NewOrFilter(
			NewContainsCondition("tags", "sale"),
			NewEqualsCondition("featured", true),
		),
	)
	
	// Convert to JSON
	jsonMap := filter.ToJSON()
	
	// Basic structure checks
	if jsonMap["operator"] != "and" {
		t.Errorf("Expected operator 'and', got %v", jsonMap["operator"])
	}
	
	conditions, ok := jsonMap["conditions"].([]interface{})
	if !ok {
		t.Fatalf("Expected conditions to be an array")
	}
	
	if len(conditions) != 3 {
		t.Errorf("Expected 3 conditions, got %d", len(conditions))
	}
	
	// Check for proper structure of children
	// This is a simple check - you could add more detailed verification
	found := false
	for _, c := range conditions {
		if cond, ok := c.(map[string]interface{}); ok {
			if cond["operator"] == "or" {
				found = true
				break
			}
		}
	}
	
	if !found {
		t.Errorf("Could not find nested OR operator in JSON output")
	}
	
	// Try recreating filter from JSON and verify it works the same
	recreated := FilterFromJSON(jsonMap)
	
	// Test with a matching metadata
	testData := map[string]interface{}{
		"category": "electronics",
		"price":    500,
		"tags":     []string{"sale", "featured"},
	}
	
	if !recreated.Matches(testData) {
		t.Errorf("Recreated filter from JSON should match valid data")
	}
	
	// Test with non-matching data
	badData := map[string]interface{}{
		"category": "clothing",
		"price":    500,
	}
	
	if recreated.Matches(badData) {
		t.Errorf("Recreated filter shouldn't match invalid data")
	}
}

func TestGetDeepValue(t *testing.T) {
	// Test metadata with nested structure
	metadata := map[string]interface{}{
		"top": "level",
		"user": map[string]interface{}{
			"name": "John",
			"profile": map[string]interface{}{
				"age":  30,
				"role": "admin",
			},
		},
		"tags": []string{"one", "two", "three"},
		"scores": []interface{}{
			map[string]interface{}{"subject": "math", "value": 90},
			map[string]interface{}{"subject": "science", "value": 85},
		},
	}
	
	// Test cases
	testCases := []struct {
		path     string
		expected interface{}
		ok       bool
	}{
		{"top", "level", true},
		{"user.name", "John", true},
		{"user.profile.age", 30, true},
		{"user.profile.role", "admin", true},
		{"tags", []string{"one", "two", "three"}, true},
		{"scores.0.subject", "math", true},
		{"scores.1.value", 85, true},
		
		// Paths that don't exist
		{"missing", nil, false},
		{"user.missing", nil, false},
		{"user.profile.missing", nil, false},
		{"tags.5", nil, false},        // Out of bounds
		{"scores.5.value", nil, false}, // Out of bounds
	}
	
	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			value, ok := GetDeepValue(metadata, tc.path)
			
			if ok != tc.ok {
				t.Errorf("GetDeepValue(%s) existence = %v, want %v", tc.path, ok, tc.ok)
			}
			
			if tc.ok {
				// For arrays/slices, can't use reflect.DeepEqual directly on interface{}
				// due to type conversion, so we check element by element
				if arr, ok := tc.expected.([]string); ok {
					if valArr, ok := value.([]string); ok {
						if len(arr) != len(valArr) {
							t.Errorf("GetDeepValue(%s) = array of length %d, want array of length %d",
								tc.path, len(valArr), len(arr))
						}
						for i, v := range arr {
							if i < len(valArr) && v != valArr[i] {
								t.Errorf("GetDeepValue(%s)[%d] = %v, want %v", tc.path, i, valArr[i], v)
							}
						}
					} else if valArr, ok := value.([]interface{}); ok {
						if len(arr) != len(valArr) {
							t.Errorf("GetDeepValue(%s) = array of length %d, want array of length %d",
								tc.path, len(valArr), len(arr))
						}
						for i, v := range arr {
							if i < len(valArr) && v != valArr[i] {
								t.Errorf("GetDeepValue(%s)[%d] = %v, want %v", tc.path, i, valArr[i], v)
							}
						}
					} else {
						t.Errorf("GetDeepValue(%s) = %v (type %T), want string array %v",
							tc.path, value, value, tc.expected)
					}
				} else if !reflect.DeepEqual(value, tc.expected) {
					t.Errorf("GetDeepValue(%s) = %v (type %T), want %v (type %T)",
						tc.path, value, value, tc.expected, tc.expected)
				}
			}
		})
	}
}