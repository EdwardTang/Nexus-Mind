package vectorstore

import (
	"crypto/md5"
	"encoding/binary"
	"sort"
	"strconv"
	"sync"
)

// TokenRing implements a consistent hashing ring
type TokenRing struct {
	mu                sync.RWMutex
	tokens            map[uint64]string  // token -> nodeID
	sortedTokens      []uint64           // tokens sorted for binary search
	nodeTokens        map[string][]uint64 // nodeID -> list of tokens owned by this node
	virtualNodes      int                // number of virtual nodes per physical node
	replicationFactor int                // number of nodes to replicate each vector to
}

// NewTokenRing creates a new token ring
func NewTokenRing(virtualNodes int, replicationFactor int) *TokenRing {
	return &TokenRing{
		tokens:           make(map[uint64]string),
		sortedTokens:     make([]uint64, 0),
		nodeTokens:       make(map[string][]uint64),
		virtualNodes:     virtualNodes,
		replicationFactor: replicationFactor,
	}
}

// AddNode adds a node to the token ring
func (r *TokenRing) AddNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if node already exists
	if _, exists := r.nodeTokens[nodeID]; exists {
		return
	}
	
	// Create virtual nodes
	nodeTokens := make([]uint64, 0, r.virtualNodes)
	for i := 0; i < r.virtualNodes; i++ {
		// Fixed: Use strconv.Itoa for proper string conversion
		token := r.hashKey(nodeID + ":" + strconv.Itoa(i))
		r.tokens[token] = nodeID
		nodeTokens = append(nodeTokens, token)
	}
	
	r.nodeTokens[nodeID] = nodeTokens
	
	// Update sorted tokens
	r.updateSortedTokens()
}

// RemoveNode removes a node from the token ring
func (r *TokenRing) RemoveNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if node exists
	tokens, exists := r.nodeTokens[nodeID]
	if !exists {
		return
	}
	
	// Remove node's tokens
	for _, token := range tokens {
		delete(r.tokens, token)
	}
	
	delete(r.nodeTokens, nodeID)
	
	// Update sorted tokens
	r.updateSortedTokens()
}

// updateSortedTokens rebuilds the sorted token list
// Caller must hold the write lock
func (r *TokenRing) updateSortedTokens() {
	r.sortedTokens = make([]uint64, 0, len(r.tokens))
	for token := range r.tokens {
		r.sortedTokens = append(r.sortedTokens, token)
	}
	sort.Slice(r.sortedTokens, func(i, j int) bool {
		return r.sortedTokens[i] < r.sortedTokens[j]
	})
}

// GetNodesForVector returns the nodes responsible for a vector
func (r *TokenRing) GetNodesForVector(vectorID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if len(r.sortedTokens) == 0 {
		return []string{}
	}
	
	// Hash the vector ID
	hash := r.hashKey(vectorID)
	
	// Find the first token >= hash
	idx := sort.Search(len(r.sortedTokens), func(i int) bool {
		return r.sortedTokens[i] >= hash
	})
	
	// If we didn't find a token >= hash, wrap around to the first token
	if idx == len(r.sortedTokens) {
		idx = 0
	}
	
	// Get R distinct nodes, starting from the found position
	result := make([]string, 0, r.replicationFactor)
	visited := make(map[string]bool)
	
	// Loop until we have enough replicas or have tried all nodes
	startIdx := idx
	for len(result) < r.replicationFactor && len(visited) < len(r.nodeTokens) {
		nodeID := r.tokens[r.sortedTokens[idx]]
		if !visited[nodeID] {
			result = append(result, nodeID)
			visited[nodeID] = true
		}
		
		idx = (idx + 1) % len(r.sortedTokens)
		
		// Safety check to avoid infinite loop if replicationFactor > node count
		if idx == startIdx && len(result) < r.replicationFactor {
			break
		}
	}
	
	return result
}

// GetNodeDistribution returns a map of nodeID -> count of tokens owned
func (r *TokenRing) GetNodeDistribution() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]int)
	for nodeID, tokens := range r.nodeTokens {
		result[nodeID] = len(tokens)
	}
	
	return result
}

// hashKey creates a uint64 hash of a key
func (r *TokenRing) hashKey(key string) uint64 {
	hash := md5.Sum([]byte(key))
	return binary.LittleEndian.Uint64(hash[:8])
}

// GetAllNodes returns a list of all nodes in the ring
func (r *TokenRing) GetAllNodes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	nodes := make([]string, 0, len(r.nodeTokens))
	for nodeID := range r.nodeTokens {
		nodes = append(nodes, nodeID)
	}
	
	return nodes
}

// GetTokensForNode returns the tokens owned by a node
func (r *TokenRing) GetTokensForNode(nodeID string) []uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tokens, exists := r.nodeTokens[nodeID]
	if !exists {
		return []uint64{}
	}
	
	result := make([]uint64, len(tokens))
	copy(result, tokens)
	
	return result
}

// FindAffectedVectors identifies vectors that need to be moved when a node joins or leaves
// For large vector sets, consider processing in batches to reduce memory pressure
func (r *TokenRing) FindAffectedVectors(oldRing *TokenRing, vectors []string) map[string][]string {
	// Map of nodeID -> list of vectors that this node should now handle
	result := make(map[string][]string)
	
	for _, vectorID := range vectors {
		oldNodes := oldRing.GetNodesForVector(vectorID)
		newNodes := r.GetNodesForVector(vectorID)
		
		// Find nodes that should now have this vector but didn't before
		for _, newNode := range newNodes {
			found := false
			for _, oldNode := range oldNodes {
				if newNode == oldNode {
					found = true
					break
				}
			}
			
			if !found {
				if result[newNode] == nil {
					result[newNode] = make([]string, 0)
				}
				result[newNode] = append(result[newNode], vectorID)
			}
		}
	}
	
	return result
}

// Clone creates a deep copy of the token ring
func (r *TokenRing) Clone() *TokenRing {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	clone := NewTokenRing(r.virtualNodes, r.replicationFactor)
	
	// Copy tokens
	for token, nodeID := range r.tokens {
		clone.tokens[token] = nodeID
	}
	
	// Copy sortedTokens
	clone.sortedTokens = make([]uint64, len(r.sortedTokens))
	copy(clone.sortedTokens, r.sortedTokens)
	
	// Copy nodeTokens
	for nodeID, tokens := range r.nodeTokens {
		nodeTokensCopy := make([]uint64, len(tokens))
		copy(nodeTokensCopy, tokens)
		clone.nodeTokens[nodeID] = nodeTokensCopy
	}
	
	return clone
}