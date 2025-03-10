package models

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// Vector represents a high-dimensional embedding
type Vector struct {
	ID        string                 // Unique identifier
	Values    []float32              // Vector values (fixed dimensions per collection)
	Metadata  map[string]interface{} // Optional associated metadata
	Timestamp int64                  // Creation/modification timestamp
	Deleted   bool                   // Soft deletion marker
}

// SparseVector represents a sparse vector with explicit indices and values
type SparseVector struct {
	ID        string                 // Unique identifier
	Indices   []int                  // Indices of non-zero elements
	Values    []float32              // Values at those indices
	Metadata  map[string]interface{} // Optional associated metadata
	Timestamp int64                  // Creation/modification timestamp
	Deleted   bool                   // Soft deletion marker
}

// NewVector creates a new dense vector with the current timestamp
func NewVector(id string, values []float32, metadata map[string]interface{}) *Vector {
	return &Vector{
		ID:        id,
		Values:    values,
		Metadata:  metadata,
		Timestamp: time.Now().UnixNano(),
		Deleted:   false,
	}
}

// NewSparseVector creates a new sparse vector with the current timestamp
func NewSparseVector(id string, indices []int, values []float32, metadata map[string]interface{}) *SparseVector {
	return &SparseVector{
		ID:        id,
		Indices:   indices,
		Values:    values,
		Metadata:  metadata,
		Timestamp: time.Now().UnixNano(),
		Deleted:   false,
	}
}

// Copy creates a deep copy of the vector
func (v *Vector) Copy() *Vector {
	valuesCopy := make([]float32, len(v.Values))
	copy(valuesCopy, v.Values)

	metadataCopy := make(map[string]interface{})
	for k, v := range v.Metadata {
		metadataCopy[k] = v
	}

	return &Vector{
		ID:        v.ID,
		Values:    valuesCopy,
		Metadata:  metadataCopy,
		Timestamp: v.Timestamp,
		Deleted:   v.Deleted,
	}
}

// Normalize normalizes the vector in place (L2 norm)
func (v *Vector) Normalize() {
	var sum float32
	for _, val := range v.Values {
		sum += val * val
	}
	
	if sum == 0 {
		return
	}
	
	magnitude := float32(math.Sqrt(float64(sum)))
	for i := range v.Values {
		v.Values[i] /= magnitude
	}
}

// Dimension returns the dimensionality of the vector
func (v *Vector) Dimension() int {
	return len(v.Values)
}

// Mark the vector as deleted (soft deletion)
func (v *Vector) MarkDeleted() {
	v.Deleted = true
	v.Timestamp = time.Now().UnixNano()
}

// Size returns the approximate memory size of the vector in bytes
func (v *Vector) Size() int {
	// Base size: ID (string pointer + length) + slice header + timestamp + deleted flag
	size := 16 + 24 + 8 + 1
	
	// Add size of the vector values
	size += len(v.Values) * 4 // float32 = 4 bytes
	
	// Add size of the metadata (rough estimate)
	metadataSize := 0
	for k, val := range v.Metadata {
		metadataSize += len(k)
		
		// Rough estimate for different types
		switch v := val.(type) {
		case string:
			metadataSize += len(v)
		case float64:
			metadataSize += 8
		case int:
			metadataSize += 8
		case bool:
			metadataSize += 1
		default:
			metadataSize += 8 // Default estimate for unknown types
		}
	}
	
	return size + metadataSize
}

// Serialize converts the vector to a byte array for persistence
func (v *Vector) Serialize() []byte {
	// This is a simplified serialization - in production we would use
	// a more sophisticated approach or a library like Protocol Buffers
	
	// Calculate total size
	idBytes := []byte(v.ID)
	metadataBytes := serializeMetadata(v.Metadata)
	
	// ID length (4) + ID + Values length (4) + Values + 
	// Metadata length (4) + Metadata + Timestamp (8) + Deleted (1)
	totalSize := 4 + len(idBytes) + 4 + len(v.Values)*4 + 4 + len(metadataBytes) + 8 + 1
	
	buf := make([]byte, totalSize)
	offset := 0
	
	// Write ID
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(idBytes)))
	offset += 4
	copy(buf[offset:], idBytes)
	offset += len(idBytes)
	
	// Write Values
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(v.Values)))
	offset += 4
	for _, val := range v.Values {
		binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(val))
		offset += 4
	}
	
	// Write Metadata
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(metadataBytes)))
	offset += 4
	copy(buf[offset:], metadataBytes)
	offset += len(metadataBytes)
	
	// Write Timestamp
	binary.LittleEndian.PutUint64(buf[offset:], uint64(v.Timestamp))
	offset += 8
	
	// Write Deleted flag
	if v.Deleted {
		buf[offset] = 1
	} else {
		buf[offset] = 0
	}
	
	return buf
}

// Deserialize constructs a vector from its serialized form
func DeserializeVector(data []byte) (*Vector, error) {
	if len(data) < 4 {
		return nil, ErrInvalidFormat
	}
	
	offset := 0
	
	// Read ID
	idLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	if offset+int(idLen) > len(data) {
		return nil, ErrInvalidFormat
	}
	id := string(data[offset : offset+int(idLen)])
	offset += int(idLen)
	
	// Read Values
	if offset+4 > len(data) {
		return nil, ErrInvalidFormat
	}
	valuesLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	if offset+int(valuesLen)*4 > len(data) {
		return nil, ErrInvalidFormat
	}
	
	values := make([]float32, valuesLen)
	for i := 0; i < int(valuesLen); i++ {
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
	}
	
	// Read Metadata
	if offset+4 > len(data) {
		return nil, ErrInvalidFormat
	}
	metadataLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	if offset+int(metadataLen) > len(data) {
		return nil, ErrInvalidFormat
	}
	
	metadata, err := deserializeMetadata(data[offset : offset+int(metadataLen)])
	if err != nil {
		return nil, err
	}
	offset += int(metadataLen)
	
	// Read Timestamp
	if offset+8 > len(data) {
		return nil, ErrInvalidFormat
	}
	timestamp := int64(binary.LittleEndian.Uint64(data[offset:]))
	offset += 8
	
	// Read Deleted flag
	if offset+1 > len(data) {
		return nil, ErrInvalidFormat
	}
	deleted := data[offset] == 1
	
	return &Vector{
		ID:        id,
		Values:    values,
		Metadata:  metadata,
		Timestamp: timestamp,
		Deleted:   deleted,
	}, nil
}

// serializeMetadata converts metadata to a byte array
// This is a simplified implementation - in production you would use
// a more sophisticated approach
func serializeMetadata(metadata map[string]interface{}) []byte {
	// Simplified implementation - just a placeholder
	// In a real implementation, we would properly encode the structure
	return []byte{}
}

// deserializeMetadata reconstructs metadata from its serialized form
func deserializeMetadata(data []byte) (map[string]interface{}, error) {
	// Simplified implementation - just a placeholder
	// In a real implementation, we would properly decode the structure
	return map[string]interface{}{}, nil
}

// Common errors for serialization
var (
	ErrInvalidFormat = fmt.Errorf("invalid vector format")
)