package models

import (
	"fmt"
	"reflect"
	"strings"
)

// FieldType represents the data type of a metadata field
type FieldType int

const (
	StringField FieldType = iota
	NumberField
	BoolField
	ArrayField
	GeoField
)

// MetadataSchema defines typed fields for efficient filtering
type MetadataSchema struct {
	Fields map[string]FieldType
}

// NewMetadataSchema creates a new empty metadata schema
func NewMetadataSchema() *MetadataSchema {
	return &MetadataSchema{
		Fields: make(map[string]FieldType),
	}
}

// AddField adds a field to the schema with the specified type
func (s *MetadataSchema) AddField(name string, fieldType FieldType) {
	s.Fields[name] = fieldType
}

// ValidateMetadata checks if the provided metadata conforms to the schema
func (s *MetadataSchema) ValidateMetadata(metadata map[string]interface{}) error {
	for name, expectedType := range s.Fields {
		value, exists := metadata[name]
		if !exists {
			continue // Field is optional
		}

		// Validate type
		actualType := detectFieldType(value)
		if actualType != expectedType {
			return fmt.Errorf("field %s has wrong type: expected %v, got %v", name, expectedType, actualType)
		}
	}
	return nil
}

// detectFieldType determines the FieldType based on a Go value
func detectFieldType(value interface{}) FieldType {
	if value == nil {
		return StringField // Default
	}

	switch v := value.(type) {
	case string:
		return StringField
	case float32, float64, int, int32, int64, uint, uint32, uint64:
		return NumberField
	case bool:
		return BoolField
	case []interface{}, []string, []int, []float64:
		return ArrayField
	default:
		// Check if it's a geo point (assume map with lat/lon)
		if m, ok := v.(map[string]interface{}); ok {
			if _, hasLat := m["lat"]; hasLat {
				if _, hasLon := m["lon"]; hasLon {
					return GeoField
				}
			}
		}
		return StringField // Default fallback
	}
}

// FilterOperator defines how multiple conditions are combined
type FilterOperator int

const (
	AND FilterOperator = iota
	OR
)

// FilterCondition represents a single filtering condition
type FilterCondition struct {
	Field    string      // Path to the field
	Operator string      // eq, gt, lt, range, contains
	Value    interface{} // Value to compare against
}

// NewEqualsCondition creates a condition that checks for equality
func NewEqualsCondition(field string, value interface{}) FilterCondition {
	return FilterCondition{
		Field:    field,
		Operator: "eq",
		Value:    value,
	}
}

// NewRangeCondition creates a condition that checks if a value is within a range
func NewRangeCondition(field string, min, max interface{}) FilterCondition {
	return FilterCondition{
		Field:    field,
		Operator: "range",
		Value: map[string]interface{}{
			"gte": min,
			"lte": max,
		},
	}
}

// MetadataFilter represents a filter for metadata based on conditions
type MetadataFilter struct {
	Conditions []FilterCondition
	Operator   FilterOperator // AND or OR
}

// NewAndFilter creates a filter that combines conditions with AND logic
func NewAndFilter(conditions ...FilterCondition) *MetadataFilter {
	return &MetadataFilter{
		Conditions: conditions,
		Operator:   AND,
	}
}

// NewOrFilter creates a filter that combines conditions with OR logic
func NewOrFilter(conditions ...FilterCondition) *MetadataFilter {
	return &MetadataFilter{
		Conditions: conditions,
		Operator:   OR,
	}
}

// MatchVector checks if a vector's metadata matches the filter
func (f *MetadataFilter) MatchVector(vector *Vector) bool {
	if f == nil || len(f.Conditions) == 0 {
		return true // Empty filter matches everything
	}

	// If metadata is nil, only match if the filter is empty
	if vector.Metadata == nil {
		return len(f.Conditions) == 0
	}

	if f.Operator == AND {
		// All conditions must match
		for _, condition := range f.Conditions {
			if !matchCondition(vector.Metadata, condition) {
				return false
			}
		}
		return true
	} else {
		// At least one condition must match
		for _, condition := range f.Conditions {
			if matchCondition(vector.Metadata, condition) {
				return true
			}
		}
		return false
	}
}

// matchCondition checks if metadata matches a specific condition
func matchCondition(metadata map[string]interface{}, condition FilterCondition) bool {
	// Extract the value from metadata
	pathParts := strings.Split(condition.Field, ".")
	value := getNestedValue(metadata, pathParts)
	
	if value == nil {
		// Field doesn't exist
		return false
	}

	switch condition.Operator {
	case "eq":
		return reflect.DeepEqual(value, condition.Value)
	case "neq":
		return !reflect.DeepEqual(value, condition.Value)
	case "gt":
		return compareValues(value, condition.Value) > 0
	case "gte":
		return compareValues(value, condition.Value) >= 0
	case "lt":
		return compareValues(value, condition.Value) < 0
	case "lte":
		return compareValues(value, condition.Value) <= 0
	case "range":
		if rangeValues, ok := condition.Value.(map[string]interface{}); ok {
			min, hasMin := rangeValues["gte"]
			max, hasMax := rangeValues["lte"]
			
			if hasMin && compareValues(value, min) < 0 {
				return false
			}
			if hasMax && compareValues(value, max) > 0 {
				return false
			}
			return true
		}
		return false
	case "contains":
		if strVal, ok := value.(string); ok {
			if condStrVal, ok := condition.Value.(string); ok {
				return strings.Contains(strVal, condStrVal)
			}
		} else if arrVal, ok := value.([]interface{}); ok {
			for _, item := range arrVal {
				if reflect.DeepEqual(item, condition.Value) {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

// getNestedValue retrieves a value from nested maps using a path
func getNestedValue(data map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return nil
	}

	current := data[path[0]]
	if len(path) == 1 {
		return current
	}

	if nestedMap, ok := current.(map[string]interface{}); ok {
		return getNestedValue(nestedMap, path[1:])
	}

	return nil
}

// compareValues compares two values and returns:
// -1 if v1 < v2
//  0 if v1 == v2
//  1 if v1 > v2
func compareValues(v1, v2 interface{}) int {
	// Handle nil cases
	if v1 == nil && v2 == nil {
		return 0
	}
	if v1 == nil {
		return -1
	}
	if v2 == nil {
		return 1
	}

	// Compare based on type
	switch val1 := v1.(type) {
	case string:
		if val2, ok := v2.(string); ok {
			if val1 < val2 {
				return -1
			} else if val1 > val2 {
				return 1
			}
			return 0
		}
	case float64:
		switch val2 := v2.(type) {
		case float64:
			if val1 < val2 {
				return -1
			} else if val1 > val2 {
				return 1
			}
			return 0
		case int:
			val2Float := float64(val2)
			if val1 < val2Float {
				return -1
			} else if val1 > val2Float {
				return 1
			}
			return 0
		}
	case int:
		switch val2 := v2.(type) {
		case int:
			if val1 < val2 {
				return -1
			} else if val1 > val2 {
				return 1
			}
			return 0
		case float64:
			val1Float := float64(val1)
			if val1Float < val2 {
				return -1
			} else if val1Float > val2 {
				return 1
			}
			return 0
		}
	}

	// Default: if types don't match or can't be compared
	return 0
}