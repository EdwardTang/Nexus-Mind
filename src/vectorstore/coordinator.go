package vectorstore

import (
	"fmt"
	"sync"
	"time"
)

// Operation status constants
const (
	OpPending   = "pending"
	OpRunning   = "running"
	OpCompleted = "completed"
	OpFailed    = "failed"
)

// RebalanceConfig holds configuration for the coordinator's rebalancing operations
type RebalanceConfig struct {
	BatchSize            int           `json:"batchSize"`
	MaxConcurrentTransfers int          `json:"maxConcurrentTransfers"`
	TaskTimeout          time.Duration `json:"taskTimeout"`
}

// DefaultRebalanceConfig returns a default rebalance configuration
func DefaultRebalanceConfig() RebalanceConfig {
	return RebalanceConfig{
		BatchSize:            100,
		MaxConcurrentTransfers: 3,
		TaskTimeout:          5 * time.Minute,
	}
}

// RebalanceOperation represents a rebalancing operation
type RebalanceOperation struct {
	ID          string                 `json:"id"`
	StartTime   time.Time              `json:"startTime"`
	EndTime     time.Time              `json:"endTime,omitempty"`
	Status      string                 `json:"status"`
	Events      []ClusterChangeEvent   `json:"events"`
	TaskCount   int                    `json:"taskCount"`
	CompletedTasks int                 `json:"completedTasks"`
	FailedTasks int                    `json:"failedTasks"`
	LastError   string                 `json:"lastError,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// Coordinator manages vector distribution and rebalancing
type Coordinator struct {
	config            RebalanceConfig
	logger            Logger
	transferService   *TransferService
	vectorStore       *VectorStore
	tokenRing         *TokenRing
	membershipService *MembershipService
	operations        map[string]*RebalanceOperation
	currentMetrics    map[string]interface{}
	mu                sync.RWMutex
}

// NewCoordinator creates a new coordinator instance
func NewCoordinator(config RebalanceConfig, logger Logger) *Coordinator {
	return &Coordinator{
		config:         config,
		logger:         logger,
		operations:     make(map[string]*RebalanceOperation),
		currentMetrics: make(map[string]interface{}),
	}
}

// SetServices sets the services used by the coordinator
func (c *Coordinator) SetServices(membership *MembershipService, transfer *TransferService, store *VectorStore, ring *TokenRing) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.membershipService = membership
	c.transferService = transfer
	c.vectorStore = store
	c.tokenRing = ring
	
	c.logger.Info("Coordinator services initialized")
}

// TriggerRebalancing starts a rebalancing operation based on cluster events
func (c *Coordinator) TriggerRebalancing(events []ClusterChangeEvent) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Create a new operation
	opID := fmt.Sprintf("rebalance-%d", time.Now().UnixNano())
	op := &RebalanceOperation{
		ID:        opID,
		StartTime: time.Now(),
		Status:    OpPending,
		Events:    events,
	}
	
	c.operations[opID] = op
	
	// In a real implementation, this would start the rebalancing process
	// For now, just set it as completed
	op.Status = OpCompleted
	op.EndTime = time.Now()
	op.Metrics = map[string]interface{}{
		"duration": op.EndTime.Sub(op.StartTime).String(),
		"eventCount": len(events),
	}
	
	c.currentMetrics = op.Metrics
	
	c.logger.Info("Triggered rebalancing operation %s for %d events", opID, len(events))
	return opID
}

// GetAllOperations returns all rebalancing operations
func (c *Coordinator) GetAllOperations() []*RebalanceOperation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	operations := make([]*RebalanceOperation, 0, len(c.operations))
	for _, op := range c.operations {
		operations = append(operations, op)
	}
	
	return operations
}

// GetOperation returns a specific operation by ID
func (c *Coordinator) GetOperation(id string) (*RebalanceOperation, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	op, exists := c.operations[id]
	return op, exists
}

// GetCurrentMetrics returns the current rebalancing metrics
func (c *Coordinator) GetCurrentMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.currentMetrics
}