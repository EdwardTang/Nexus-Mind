# Design Document: Automated Scaling & Load Balancing

## Overview
This document outlines the design for implementing basic automated shard rebalancing when nodes join or leave the vector store cluster.

## Goals
- Detect cluster membership changes automatically
- Redistribute vector data shards to maintain balance
- Minimize disruption to ongoing operations during rebalancing
- Keep implementation simple and aligned with MVP architecture

## Non-Goals
- Complex multi-tenant balancing algorithms
- Capacity-based or workload-based balancing
- Hot-spot detection and mitigation
- Zero-downtime node removal

## System Components

### 1. Membership Change Detection

**Implementation:**
- Leverage the existing Raft consensus mechanism to detect node changes
- Extend the Raft state machine to track cluster membership events
- Add a `ClusterChangeEvent` type to the Raft log with:
  ```go
  type ChangeType int
  const (
    NodeJoined ChangeType = iota
    NodeLeft
  )
  
  type ClusterChangeEvent struct {
    Type      ChangeType
    NodeID    string
    Timestamp int64
  }
  ```

**Behavior:**
- When a node joins: The node registers with the leader via RPC
- When a node leaves: Either clean shutdown or failure detection via missed heartbeats
- Leader commits the change event to the Raft log
- All nodes process the membership change and update their local view

**Stabilization Period:**
- Implement a configurable cooldown period after membership changes:
  ```go
  type MembershipConfig struct {
    StabilizationPeriodSeconds int  // Default: 30 seconds
    MaxPendingEvents          int  // Maximum queued membership events
  }
  ```
- Only trigger rebalancing after the stabilization period to prevent thrashing
- Coalesce multiple rapid membership changes into a single rebalancing operation

### 2. Partition Assignment Algorithm

**Implementation:**
- Use consistent hashing for shard distribution
- Each vector is assigned to `R` nodes where `R` is the replication factor
- Define a token ring where each node owns multiple tokens

```go
type TokenRing struct {
  tokens map[uint64]string  // token -> nodeID
}

func (r *TokenRing) GetNodesForVector(vectorID string) []string {
  hash := hash(vectorID)
  // Find R nodes by walking the token ring
  // Return node IDs responsible for this vector
}
```

**Rebalancing Logic:**
- When a node joins:
  - Assign tokens to the new node based on an equal distribution
  - Identify which existing tokens/shards need to move
- When a node leaves:
  - Reassign its tokens to remaining nodes
  - Identify which shards need to be replicated to maintain replication factor

### 3. Data Movement Service

**Implementation:**
- Create a background worker that executes rebalancing operations
- Use a priority queue to manage transfer tasks:

```go
type TransferTask struct {
  SourceNodeID   string
  DestNodeID     string
  ShardID        string
  Priority       int
  State          TaskState
  VectorIDs      []string
  AttemptCount   int        // Number of retry attempts
  LastError      string     // Last error message if failed
  CreatedAt      int64      // Creation timestamp
  UpdatedAt      int64      // Last update timestamp
}

type TaskState int
const (
  Pending TaskState = iota
  InProgress
  Completed
  Failed
  Retrying
)
```

**Retry Logic:**
```go
type RetryConfig struct {
  MaxRetries          int     // Maximum retry attempts per task
  InitialBackoffMs    int     // Initial backoff in milliseconds
  BackoffMultiplier   float32 // Multiplier for exponential backoff
  MaxBackoffMs        int     // Maximum backoff in milliseconds
  JitterFactor        float32 // Random jitter factor (0.0-1.0)
}

// Default values:
// MaxRetries: 3
// InitialBackoffMs: 1000 (1 second)
// BackoffMultiplier: 2.0
// MaxBackoffMs: 30000 (30 seconds)
// JitterFactor: 0.2
```

**Data Transfer Protocol:**
1. Source node receives transfer request for a shard
2. Source streams vector data to destination node in batches
3. Destination builds local HNSW index as data arrives
4. On completion, destination acknowledges receipt
5. Coordinator updates metadata once transfer completes

**Partial Failure Handling:**
- Divide large shards into sub-tasks for more granular success/failure
- Implement a transaction log for tracking sub-task status:
  ```go
  type SubTask struct {
    TaskID       string
    SegmentID    string
    VectorRange  [2]string  // Start/end vector IDs in segment
    State        TaskState
    BytesTotal   int64
    BytesMoved   int64
  }
  ```
- Allow individual sub-tasks to succeed/fail independently
- Only mark a shard transfer as complete when all sub-tasks succeed

### 4. Throttling & Monitoring

**Implementation:**
- Add configurable throttling to limit resource usage during rebalancing:

```go
type RebalanceConfig struct {
  MaxConcurrentTransfers  int   // Default: 3
  MaxBandwidthPerTransfer int   // KB/s, Default: 5120 (5MB/s)
  BatchSize               int   // vectors per batch, Default: 1000
  MaxCpuPercent           int   // Maximum CPU usage percent, Default: 50
  MaxMemoryPercent        int   // Maximum memory usage percent, Default: 70
}
```

- Track rebalancing progress with metrics:

```go
type RebalanceMetrics struct {
  TotalShardsToMove    int
  ShardsCompleted      int
  BytesTransferred     int64
  StartTime            int64
  EstimatedCompletion  int64
  CurrentThroughput    int    // vectors/second
  QueryLatencyImpact   float32 // percentage increase in p99 latency
  FailedTasks          int
  RetryingTasks        int
}
```

- Add performance impact tracking:
  ```go
  type PerformanceSnapshot struct {
    Timestamp       int64
    QueryLatencyP50 int  // milliseconds
    QueryLatencyP99 int  // milliseconds
    QueriesPerSecond int
    NodeCpuPercent  map[string]float32  // nodeID -> CPU usage %
    NodeMemPercent  map[string]float32  // nodeID -> memory usage %
  }
  ```

- Implement adaptive throttling based on performance impact:
  - If query latency increases beyond threshold, reduce transfer rate
  - If node resource usage exceeds safe limits, pause transfers temporarily

### 5. Metadata Updates

**Implementation:**
- Use Raft to commit metadata changes once transfers complete
- Define a transaction log for atomic metadata updates:

```go
type MetadataTransaction struct {
  TransactionID string
  ShardID       string
  OldNodeIDs    []string
  NewNodeIDs    []string
  Timestamp     int64
  Status        TransactionStatus
  SubTaskStatus map[string]TaskState  // Tracks sub-task completion
}

type TransactionStatus int
const (
  Prepared TransactionStatus = iota
  Committed
  RolledBack
  PartiallyCommitted
)
```

- Support partial metadata commits:
  - If all sub-tasks for a shard transfer succeed, commit normally
  - If only some sub-tasks succeed, update metadata for just those segments
  - Maintain a "needs repair" flag for partially moved shards

### 6. Rollback & Recovery

**Implementation:**
- Add transaction logging to track progress of each rebalancing operation
- Implement a recovery mechanism for interrupted operations:
  ```go
  type RebalanceOperation struct {
    OperationID   string
    TriggerEvent  ClusterChangeEvent
    StartTime     int64
    Status        OperationStatus
    Tasks         []string  // List of task IDs
    MetadataState string    // Serialized state for recovery
  }
  ```
- Store checkpoint data periodically to enable recovery after node restarts

## Operation Flow

1. **Cluster Change Detection:**
   - Node joins/leaves cluster
   - Raft consensus registers the membership change
   - Wait for stabilization period to prevent thrashing
   - Coordinator node initiates rebalancing

2. **Rebalancing Planning:**
   - Calculate new token assignments
   - Determine which shards need to move and where
   - Create and prioritize transfer tasks
   - Divide large shards into sub-tasks for better granularity

3. **Data Movement:**
   - Execute transfer tasks with throttling and resource monitoring
   - Source nodes stream vectors to destination nodes
   - Destination nodes build indices incrementally
   - Track resource usage on all nodes during index building
   - Implement exponential backoff for failed transfers

4. **Metadata Update:**
   - Once transfers complete, prepare metadata transaction
   - Allow partial commits for partially successful transfers
   - Commit metadata changes via Raft
   - Update routing tables on all nodes
   - Mark rebalancing operation as complete

5. **Verification & Cleanup:**
   - Validate all shards are accessible on new nodes
   - Ensure query routing uses updated metadata
   - Clean up temporary transfer data
   - Schedule repairs for any partially transferred shards

## Success Metrics

- **Balance Achieved:** No node is responsible for >20% more shards than average
- **Rebalance Time:** Complete within 5 minutes for cluster size changes
- **Operation Impact:** <10% increase in p99 query latency during rebalancing
- **Reliability:** 100% of shards remain available throughout rebalancing
- **Resource Safety:** Node CPU and memory usage remain below 80% during rebalancing

## Future Improvements

- Prioritize hot shard movement based on query patterns
- Add capacity-aware balancing based on node resources
- Implement incremental HNSW index updates to speed up transfers
- Add predictive scaling based on workload trends

## Implementation Plan

1. Add cluster membership event tracking to Raft layer
2. Implement consistent hashing with configurable tokens
3. Build the data transfer service with retry mechanism
4. Add metadata transaction support with partial commits
5. Implement resource monitoring and adaptive throttling
6. Create coordinator service to orchestrate rebalancing
7. Add monitoring and reporting for rebalance operations