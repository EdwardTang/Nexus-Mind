package vectorstore

import (
	"sync"
	"time"
)

// ChangeType represents the type of cluster membership change
type ChangeType int

const (
	NodeJoined ChangeType = iota
	NodeLeft
)

func (c ChangeType) String() string {
	switch c {
	case NodeJoined:
		return "NodeJoined"
	case NodeLeft:
		return "NodeLeft"
	default:
		return "Unknown"
	}
}

// ClusterChangeEvent represents a cluster membership change event
type ClusterChangeEvent struct {
	Type      ChangeType
	NodeID    string
	Timestamp int64
}

// MembershipConfig holds configuration for the membership service
type MembershipConfig struct {
	StabilizationPeriodSeconds int // Default: 30 seconds
	MaxPendingEvents          int // Maximum queued membership events
}

// DefaultMembershipConfig returns the default membership configuration
func DefaultMembershipConfig() MembershipConfig {
	return MembershipConfig{
		StabilizationPeriodSeconds: 30,
		MaxPendingEvents:           100,
	}
}

// MembershipService manages cluster membership and detects changes
type MembershipService struct {
	mu            sync.Mutex
	config        MembershipConfig
	nodes         map[string]NodeInfo
	pendingEvents []ClusterChangeEvent
	lastEventTime int64
	coordinator   *Coordinator // Reference to the rebalance coordinator
}

// NodeInfo stores information about a cluster node
type NodeInfo struct {
	ID           string
	Address      string
	Status       NodeStatus
	JoinTime     int64
	LastHeartbeat int64
}

// NodeStatus represents the status of a node
type NodeStatus int

const (
	NodeStatusUnknown NodeStatus = iota
	NodeStatusJoining
	NodeStatusActive
	NodeStatusLeaving
	NodeStatusDead
)

// NewMembershipService creates a new membership service
func NewMembershipService(config MembershipConfig) *MembershipService {
	return &MembershipService{
		config:        config,
		nodes:         make(map[string]NodeInfo),
		pendingEvents: make([]ClusterChangeEvent, 0),
		lastEventTime: time.Now().UnixNano(),
	}
}

// SetCoordinator sets the rebalance coordinator reference
func (ms *MembershipService) SetCoordinator(coordinator *Coordinator) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.coordinator = coordinator
}

// RegisterNode registers a new node in the cluster
func (ms *MembershipService) RegisterNode(nodeID string, address string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	now := time.Now().UnixNano()
	_, exists := ms.nodes[nodeID]
	
	if !exists {
		// Add the node to our local view
		ms.nodes[nodeID] = NodeInfo{
			ID:           nodeID,
			Address:      address,
			Status:       NodeStatusJoining,
			JoinTime:     now,
			LastHeartbeat: now,
		}
		
		// Create a cluster change event
		event := ClusterChangeEvent{
			Type:      NodeJoined,
			NodeID:    nodeID,
			Timestamp: now,
		}
		
		// Add to pending events
		ms.pendingEvents = append(ms.pendingEvents, event)
		ms.lastEventTime = now
		
		// TODO: Propose this event to the Raft log
		// This is a simplified version; in a complete implementation,
		// we would use the Raft consensus to ensure all nodes agree on
		// the membership change.
		
		return nil
	}
	
	// Node already exists, update its status
	node := ms.nodes[nodeID]
	node.Status = NodeStatusActive
	node.LastHeartbeat = now
	ms.nodes[nodeID] = node
	
	return nil
}

// UnregisterNode marks a node as leaving the cluster
func (ms *MembershipService) UnregisterNode(nodeID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	node, exists := ms.nodes[nodeID]
	if !exists {
		return nil // Node doesn't exist, nothing to do
	}
	
	now := time.Now().UnixNano()
	
	// Update node status
	node.Status = NodeStatusLeaving
	ms.nodes[nodeID] = node
	
	// Create a cluster change event
	event := ClusterChangeEvent{
		Type:      NodeLeft,
		NodeID:    nodeID,
		Timestamp: now,
	}
	
	// Add to pending events
	ms.pendingEvents = append(ms.pendingEvents, event)
	ms.lastEventTime = now
	
	// TODO: Propose this event to the Raft log
	
	return nil
}

// Heartbeat updates the last heartbeat time for a node
func (ms *MembershipService) Heartbeat(nodeID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	node, exists := ms.nodes[nodeID]
	if !exists {
		return nil // Node doesn't exist, nothing to do
	}
	
	node.LastHeartbeat = time.Now().UnixNano()
	ms.nodes[nodeID] = node
	
	return nil
}

// CheckStabilization checks if the cluster has stabilized and triggers rebalancing if needed
func (ms *MembershipService) CheckStabilization() bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	now := time.Now().UnixNano()
	stabilizationPeriodNanos := int64(ms.config.StabilizationPeriodSeconds) * int64(time.Second)
	
	// Check if enough time has passed since the last event
	if now - ms.lastEventTime < stabilizationPeriodNanos {
		return false // Not stabilized yet
	}
	
	// If we have pending events and the cluster has stabilized,
	// trigger rebalancing
	if len(ms.pendingEvents) > 0 && ms.coordinator != nil {
		events := make([]ClusterChangeEvent, len(ms.pendingEvents))
		copy(events, ms.pendingEvents)
		
		// Clear pending events
		ms.pendingEvents = ms.pendingEvents[:0]
		
		// Trigger rebalancing in a separate goroutine
		go ms.coordinator.TriggerRebalancing(events)
		
		return true
	}
	
	return false
}

// GetActiveNodes returns a list of currently active nodes
func (ms *MembershipService) GetActiveNodes() []NodeInfo {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	activeNodes := make([]NodeInfo, 0, len(ms.nodes))
	for _, node := range ms.nodes {
		if node.Status == NodeStatusActive || node.Status == NodeStatusJoining {
			activeNodes = append(activeNodes, node)
		}
	}
	
	return activeNodes
}

// Start starts the membership service background routines
func (ms *MembershipService) Start() {
	// Start a background goroutine to check for node failures
	go ms.checkNodeFailures()
	
	// Start a background goroutine to check for stabilization
	go ms.stabilizationChecker()
}

// Stop stops the membership service background routines
func (ms *MembershipService) Stop() {
	// TODO: Implement stopping logic
}

// Background goroutine to check for node failures
func (ms *MembershipService) checkNodeFailures() {
	for {
		time.Sleep(10 * time.Second)
		
		ms.mu.Lock()
		now := time.Now().UnixNano()
		var failedNodes []string
		
		for id, node := range ms.nodes {
			// If node hasn't sent a heartbeat in 30 seconds and is not already marked as leaving or dead
			if node.Status != NodeStatusLeaving && node.Status != NodeStatusDead {
				if now - node.LastHeartbeat > 30 * int64(time.Second) {
					failedNodes = append(failedNodes, id)
				}
			}
		}
		ms.mu.Unlock()
		
		// Handle any failed nodes
		for _, nodeID := range failedNodes {
			ms.UnregisterNode(nodeID)
		}
	}
}

// Background goroutine to check for cluster stabilization
func (ms *MembershipService) stabilizationChecker() {
	for {
		time.Sleep(5 * time.Second)
		ms.CheckStabilization()
	}
}