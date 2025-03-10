package vectorstore

import (
	"fmt"
	"sort"
	"sync"
	"testing"
)

func TestTokenRingAddNode(t *testing.T) {
	ring := NewTokenRing(10, 3)
	
	// Add a node
	ring.AddNode("node-1")
	
	// Verify the node was added
	nodes := ring.GetAllNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
	
	if nodes[0] != "node-1" {
		t.Errorf("Expected node-1, got %s", nodes[0])
	}
	
	// Verify tokens were created
	tokens := ring.GetTokensForNode("node-1")
	if len(tokens) != 10 {
		t.Errorf("Expected 10 tokens, got %d", len(tokens))
	}
	
	// Test adding the same node again (idempotent)
	ring.AddNode("node-1")
	nodes = ring.GetAllNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node after adding the same node, got %d", len(nodes))
	}
}

func TestTokenRingRemoveNode(t *testing.T) {
	ring := NewTokenRing(10, 3)
	
	// Add two nodes
	ring.AddNode("node-1")
	ring.AddNode("node-2")
	
	// Verify both nodes exist
	nodes := ring.GetAllNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
	
	// Remove one node
	ring.RemoveNode("node-1")
	
	// Verify only one node remains
	nodes = ring.GetAllNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node after removal, got %d", len(nodes))
	}
	
	if nodes[0] != "node-2" {
		t.Errorf("Expected node-2 to remain, got %s", nodes[0])
	}
	
	// Test removing a non-existent node (should be safe)
	ring.RemoveNode("nonexistent-node")
	nodes = ring.GetAllNodes()
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node after removing non-existent node, got %d", len(nodes))
	}
}

func TestGetNodesForVector(t *testing.T) {
	ring := NewTokenRing(10, 3)
	
	// Add multiple nodes
	ring.AddNode("node-1")
	ring.AddNode("node-2")
	ring.AddNode("node-3")
	ring.AddNode("node-4")
	
	// Get nodes for a vector
	vectorID := "test-vector-1"
	nodes := ring.GetNodesForVector(vectorID)
	
	// Verify we got the correct number of nodes
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes for replication, got %d", len(nodes))
	}
	
	// Verify all nodes are unique
	nodeSet := make(map[string]bool)
	for _, node := range nodes {
		nodeSet[node] = true
	}
	
	if len(nodeSet) != len(nodes) {
		t.Errorf("Expected all unique nodes, found duplicates")
	}
	
	// Test with empty vector ID
	emptyNodes := ring.GetNodesForVector("")
	if len(emptyNodes) != 3 {
		t.Errorf("Expected 3 nodes for empty vector ID, got %d", len(emptyNodes))
	}
	
	// Test with fewer nodes than replication factor
	smallRing := NewTokenRing(10, 5)
	smallRing.AddNode("node-1")
	smallRing.AddNode("node-2")
	
	smallNodes := smallRing.GetNodesForVector("test-vector")
	if len(smallNodes) > 2 {
		t.Errorf("Expected at most 2 nodes when replication factor > node count, got %d", len(smallNodes))
	}
}

func TestFindAffectedVectors(t *testing.T) {
	// Create two rings - before and after adding a node
	oldRing := NewTokenRing(10, 2)
	oldRing.AddNode("node-1")
	oldRing.AddNode("node-2")
	
	newRing := oldRing.Clone()
	newRing.AddNode("node-3")
	
	// Create some test vectors
	vectors := []string{
		"vector-1",
		"vector-2",
		"vector-3",
		"vector-4",
		"vector-5",
	}
	
	// Find affected vectors
	affected := newRing.FindAffectedVectors(oldRing, vectors)
	
	// Verify that at least some vectors will move to node-3
	node3Vectors, hasNode3 := affected["node-3"]
	if !hasNode3 {
		t.Errorf("Expected some vectors to move to node-3, but none found")
	}
	
	// Verify that we don't have vectors assigned to non-existent nodes
	for node := range affected {
		if node != "node-1" && node != "node-2" && node != "node-3" {
			t.Errorf("Unexpected node in affected vectors: %s", node)
		}
	}
	
	// Log the number of vectors moving to node-3
	if hasNode3 {
		fmt.Printf("Vectors moving to node-3: %d\n", len(node3Vectors))
	}
	
	// Test edge cases
	// Test with nil vectors
	nilResult := newRing.FindAffectedVectors(oldRing, nil)
	if len(nilResult) != 0 {
		t.Errorf("Expected empty result for nil vectors, got %d entries", len(nilResult))
	}
	
	// Test with empty vectors slice
	emptyResult := newRing.FindAffectedVectors(oldRing, []string{})
	if len(emptyResult) != 0 {
		t.Errorf("Expected empty result for empty vectors, got %d entries", len(emptyResult))
	}
}

func TestConsistentDistribution(t *testing.T) {
	// Create a ring with 5 nodes
	ring := NewTokenRing(100, 3) // Using more virtual nodes for better distribution
	
	for i := 1; i <= 5; i++ {
		ring.AddNode(fmt.Sprintf("node-%d", i))
	}
	
	// Create 1000 test vectors
	vectors := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		vectors[i] = fmt.Sprintf("test-vector-%d", i)
	}
	
	// Count vectors per node
	nodeCounts := make(map[string]int)
	
	for _, vectorID := range vectors {
		nodes := ring.GetNodesForVector(vectorID)
		for _, node := range nodes {
			nodeCounts[node]++
		}
	}
	
	// Check if distribution is roughly even
	// Each vector is replicated to 3 nodes, so we expect around 600 vectors per node
	// Allow for some variance, say +/- 15%
	expectedPerNode := 1000 * 3 / 5
	minAcceptable := int(float64(expectedPerNode) * 0.85)
	maxAcceptable := int(float64(expectedPerNode) * 1.15)
	
	for node, count := range nodeCounts {
		if count < minAcceptable || count > maxAcceptable {
			t.Errorf("Node %s has %d vectors, expected between %d and %d", 
				node, count, minAcceptable, maxAcceptable)
		}
	}
}

func TestClone(t *testing.T) {
	// Create original ring
	original := NewTokenRing(10, 3)
	original.AddNode("node-1")
	original.AddNode("node-2")
	
	// Clone the ring
	clone := original.Clone()
	
	// Verify the clone has the same nodes
	originalNodes := original.GetAllNodes()
	cloneNodes := clone.GetAllNodes()
	
	if len(originalNodes) != len(cloneNodes) {
		t.Errorf("Clone has different number of nodes: original %d, clone %d", 
			len(originalNodes), len(cloneNodes))
	}
	
	// Sort for comparison
	sort.Strings(originalNodes)
	sort.Strings(cloneNodes)
	
	for i := range originalNodes {
		if originalNodes[i] != cloneNodes[i] {
			t.Errorf("Node mismatch at index %d: original %s, clone %s", 
				i, originalNodes[i], cloneNodes[i])
		}
	}
	
	// Modify clone and verify original is unchanged
	clone.AddNode("node-3")
	
	if len(original.GetAllNodes()) != 2 {
		t.Errorf("Original ring was modified after clone was changed")
	}
	
	if len(clone.GetAllNodes()) != 3 {
		t.Errorf("Clone should have 3 nodes but has %d", len(clone.GetAllNodes()))
	}
}

func TestConcurrentNodeChanges(t *testing.T) {
	ring := NewTokenRing(10, 3)
	
	// Add several nodes concurrently
	var wg sync.WaitGroup
	nodeCount := 20
	wg.Add(nodeCount)
	
	for i := 0; i < nodeCount; i++ {
		go func(id int) {
			defer wg.Done()
			ring.AddNode(fmt.Sprintf("node-%d", id))
		}(i)
	}
	
	wg.Wait()
	
	// Verify all nodes were added
	nodes := ring.GetAllNodes()
	if len(nodes) != nodeCount {
		t.Errorf("Expected %d nodes after concurrent addition, got %d", nodeCount, len(nodes))
	}
	
	// Now remove half the nodes concurrently
	wg.Add(nodeCount / 2)
	
	for i := 0; i < nodeCount/2; i++ {
		go func(id int) {
			defer wg.Done()
			ring.RemoveNode(fmt.Sprintf("node-%d", id))
		}(i)
	}
	
	wg.Wait()
	
	// Verify half the nodes were removed
	nodes = ring.GetAllNodes()
	if len(nodes) != nodeCount/2 {
		t.Errorf("Expected %d nodes after concurrent removal, got %d", nodeCount/2, len(nodes))
	}
}

// Table-driven tests for vector assignment
func TestVectorAssignment(t *testing.T) {
	ring := NewTokenRing(50, 3)
	ring.AddNode("node-1")
	ring.AddNode("node-2")
	ring.AddNode("node-3")
	
	// Create a table of test vectors with expected assignment properties
	testCases := []struct {
		name               string
		vectorID           string
		expectedNodeCount  int
		shouldIncludeNode1 bool
	}{
		{"Standard vector", "test-vector-1", 3, true},  // Check with a value we manually verified
		{"Empty ID", "", 3, false},                     // Empty string should still hash and assign
		{"Long ID", "very-long-vector-id-" + string(make([]byte, 1000)), 3, false}, // Very long ID
		{"Unicode ID", "向量-测试", 3, false},            // Unicode characters
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodes := ring.GetNodesForVector(tc.vectorID)
			
			// Check count
			if len(nodes) != tc.expectedNodeCount {
				t.Errorf("Expected %d nodes for vector %s, got %d", 
					tc.expectedNodeCount, tc.vectorID, len(nodes))
			}
			
			// Check specific node inclusion if required
			if tc.shouldIncludeNode1 {
				found := false
				for _, node := range nodes {
					if node == "node-1" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected node-1 to be assigned to vector %s, but it wasn't", tc.vectorID)
				}
			}
		})
	}
}