package vectorstore

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// TaskState represents the state of a transfer task
type TaskState int

const (
	Pending TaskState = iota
	InProgress
	Completed
	Failed
	Retrying
)

func (s TaskState) String() string {
	switch s {
	case Pending:
		return "Pending"
	case InProgress:
		return "InProgress"
	case Completed:
		return "Completed"
	case Failed:
		return "Failed"
	case Retrying:
		return "Retrying"
	default:
		return "Unknown"
	}
}

// TransferTask represents a task to transfer vectors between nodes
type TransferTask struct {
	ID            string
	SourceNodeID  string
	DestNodeID    string
	ShardID       string
	Priority      int
	State         TaskState
	VectorIDs     []string
	AttemptCount  int        // Number of retry attempts
	LastError     string     // Last error message if failed
	CreatedAt     int64      // Creation timestamp
	UpdatedAt     int64      // Last update timestamp
	SubTasks      []*SubTask // Optional subtasks for large transfers
}

// SubTask represents a segment of a larger transfer task
type SubTask struct {
	TaskID      string
	SegmentID   string
	VectorRange [2]string // Start/end vector IDs in segment
	State       TaskState
	BytesTotal  int64
	BytesMoved  int64
}

// RetryConfig holds configuration for task retry logic
type RetryConfig struct {
	MaxRetries        int     // Maximum retry attempts per task
	InitialBackoffMs  int     // Initial backoff in milliseconds
	BackoffMultiplier float32 // Multiplier for exponential backoff
	MaxBackoffMs      int     // Maximum backoff in milliseconds
	JitterFactor      float32 // Random jitter factor (0.0-1.0)
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoffMs:  1000, // 1 second
		BackoffMultiplier: 2.0,
		MaxBackoffMs:      30000, // 30 seconds
		JitterFactor:      0.2,
	}
}

// NewTransferTask creates a new transfer task
func NewTransferTask(sourceID, destID, shardID string, vectors []string, priority int) *TransferTask {
	now := time.Now().UnixNano()
	return &TransferTask{
		ID:           fmt.Sprintf("task-%d", now),
		SourceNodeID: sourceID,
		DestNodeID:   destID,
		ShardID:      shardID,
		Priority:     priority,
		State:        Pending,
		VectorIDs:    vectors,
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateSubTasks divides a large transfer task into smaller subtasks
func (t *TransferTask) CreateSubTasks(batchSize int) {
	if len(t.VectorIDs) <= batchSize || batchSize <= 0 {
		return // No need to create subtasks
	}
	
	t.SubTasks = make([]*SubTask, 0)
	
	// Calculate number of batches needed
	batchCount := int(math.Ceil(float64(len(t.VectorIDs)) / float64(batchSize)))
	
	for i := 0; i < batchCount; i++ {
		start := i * batchSize
		end := (i + 1) * batchSize
		if end > len(t.VectorIDs) {
			end = len(t.VectorIDs)
		}
		
		// Create a range for this batch
		var startID, endID string
		if start < len(t.VectorIDs) {
			startID = t.VectorIDs[start]
		}
		if end-1 < len(t.VectorIDs) {
			endID = t.VectorIDs[end-1]
		}
		
		subTask := &SubTask{
			TaskID:      t.ID,
			SegmentID:   fmt.Sprintf("%s-seg-%d", t.ID, i),
			VectorRange: [2]string{startID, endID},
			State:       Pending,
		}
		
		t.SubTasks = append(t.SubTasks, subTask)
	}
}

// CalculateBackoff calculates the exponential backoff time for retry
func (t *TransferTask) CalculateBackoff(config RetryConfig) time.Duration {
	// Don't retry if we've hit the max
	if t.AttemptCount >= config.MaxRetries {
		return 0
	}
	
	// Calculate exponential backoff
	backoffMs := float64(config.InitialBackoffMs) * math.Pow(float64(config.BackoffMultiplier), float64(t.AttemptCount))
	
	// Cap at max backoff
	if backoffMs > float64(config.MaxBackoffMs) {
		backoffMs = float64(config.MaxBackoffMs)
	}
	
	// Add jitter
	jitter := 1.0 - float64(config.JitterFactor)/2.0 + rand.Float64()*float64(config.JitterFactor)
	backoffMs = backoffMs * jitter
	
	return time.Duration(int64(backoffMs)) * time.Millisecond
}

// TransferService manages vector data transfer between nodes
type TransferService struct {
	mu             sync.Mutex
	tasks          map[string]*TransferTask
	retryConfig    RetryConfig
	vectorStore    *VectorStore // Reference to the vector store
	maxConcurrent  int          // Maximum concurrent transfers
	activeTasks    int          // Currently active task count
	taskQueue      []*TransferTask // Priority queue for pending tasks
	maxSubTaskConcurrency int    // Maximum concurrent subtasks
	logger         Logger        // Logger interface
}

// Logger interface for transfer service logging
// Logger is already defined in logger.go
// Using the common Logger interface

// NewTransferService creates a new transfer service
func NewTransferService(retryConfig RetryConfig, maxConcurrent int, logger Logger) *TransferService {
	return &TransferService{
		tasks:                make(map[string]*TransferTask),
		retryConfig:          retryConfig,
		maxConcurrent:        maxConcurrent,
		taskQueue:            make([]*TransferTask, 0),
		maxSubTaskConcurrency: 10, // Default to 10 concurrent subtasks
		logger:               logger,
	}
}

// SetVectorStore sets the vector store reference
func (ts *TransferService) SetVectorStore(vs *VectorStore) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.vectorStore = vs
}

// QueueTask adds a task to the transfer queue
func (ts *TransferService) QueueTask(task *TransferTask) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	ts.logger.Info("Queuing transfer task %s: %s -> %s, shard %s, vectors: %d", 
		task.ID, task.SourceNodeID, task.DestNodeID, task.ShardID, len(task.VectorIDs))
	
	// Store the task
	ts.tasks[task.ID] = task
	
	// Add to priority queue
	ts.taskQueue = append(ts.taskQueue, task)
	
	// Sort queue by priority (higher values first)
	ts.sortTaskQueue()
	
	// Process queue if we have capacity
	ts.processQueue()
}

// sortTaskQueue sorts the task queue by priority (descending)
func (ts *TransferService) sortTaskQueue() {
	// Higher priority values come first
	sort.Slice(ts.taskQueue, func(i, j int) bool {
		return ts.taskQueue[i].Priority > ts.taskQueue[j].Priority
	})
}

// processQueue processes tasks from the queue up to maxConcurrent
func (ts *TransferService) processQueue() {
	// Process as many tasks as we can up to our concurrent limit
	for ts.activeTasks < ts.maxConcurrent && len(ts.taskQueue) > 0 {
		// Get the highest priority task
		task := ts.taskQueue[0]
		ts.taskQueue = ts.taskQueue[1:]
		
		// Start the task execution
		ts.activeTasks++
		ts.logger.Debug("Starting task execution: %s, active tasks: %d", task.ID, ts.activeTasks)
		go ts.executeTask(task)
	}
	
	if len(ts.taskQueue) > 0 {
		ts.logger.Debug("Tasks remaining in queue: %d", len(ts.taskQueue))
	}
}

// executeTask executes a transfer task
func (ts *TransferService) executeTask(task *TransferTask) {
	// Update task state
	ts.mu.Lock()
	task.State = InProgress
	task.AttemptCount++
	task.UpdatedAt = time.Now().UnixNano()
	ts.mu.Unlock()
	
	ts.logger.Info("Executing task %s (attempt %d of %d)", 
		task.ID, task.AttemptCount, ts.retryConfig.MaxRetries)
	
	// If we have subtasks, execute them individually
	var success bool
	if len(task.SubTasks) > 0 {
		success = ts.executeSubTasks(task)
	} else {
		// Execute single task transfer
		success = ts.executeTransfer(task)
	}
	
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	if success {
		task.State = Completed
		ts.logger.Info("Task %s completed successfully", task.ID)
	} else {
		// Check if we should retry
		if task.AttemptCount < ts.retryConfig.MaxRetries {
			task.State = Retrying
			
			// Calculate backoff time
			backoff := task.CalculateBackoff(ts.retryConfig)
			
			ts.logger.Warn("Task %s failed, retrying in %v", task.ID, backoff)
			
			// Requeue the task after backoff
			time.AfterFunc(backoff, func() {
				ts.QueueTask(task)
			})
		} else {
			task.State = Failed
			ts.logger.Error("Task %s failed permanently after %d attempts. Last error: %s", 
				task.ID, task.AttemptCount, task.LastError)
		}
	}
	
	// Update task state and free up the active task slot
	task.UpdatedAt = time.Now().UnixNano()
	ts.activeTasks--
	
	// Process more tasks if we have any in the queue
	ts.processQueue()
}

// executeSubTasks executes all subtasks in a task
func (ts *TransferService) executeSubTasks(task *TransferTask) bool {
	var wg sync.WaitGroup
	results := make([]bool, len(task.SubTasks))
	
	// Create a semaphore to limit subtask concurrency
	semaphore := make(chan struct{}, ts.maxSubTaskConcurrency)
	
	for i, subTask := range task.SubTasks {
		if subTask.State == Completed {
			results[i] = true
			continue // Skip already completed subtasks
		}
		
		wg.Add(1)
		
		// Acquire semaphore slot
		semaphore <- struct{}{}
		
		go func(i int, subTask *SubTask) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot
			
			ts.logger.Debug("Executing subtask %s (%d of %d)", 
				subTask.SegmentID, i+1, len(task.SubTasks))
			
			// Execute the subtask
			subTask.State = InProgress
			// TODO: Implement actual vector transfer for the subtask range
			
			// For demonstration, we'll simulate a successful transfer
			time.Sleep(100 * time.Millisecond)
			success := rand.Float32() > 0.1 // 90% success rate for simulation
			
			if success {
				subTask.State = Completed
				results[i] = true
				ts.logger.Debug("Subtask %s completed successfully", subTask.SegmentID)
			} else {
				subTask.State = Failed
				results[i] = false
				ts.logger.Warn("Subtask %s failed", subTask.SegmentID)
			}
		}(i, subTask)
	}
	
	wg.Wait()
	
	// Task is successful if all subtasks completed successfully
	for i, success := range results {
		if !success {
			task.LastError = fmt.Sprintf("Subtask %s failed", task.SubTasks[i].SegmentID)
			return false
		}
	}
	return true
}

// executeTransfer performs the actual vector data transfer
func (ts *TransferService) executeTransfer(task *TransferTask) bool {
	// TODO: Implement actual vector transfer logic
	// This should:
	// 1. Connect to source node
	// 2. Fetch vectors
	// 3. Send to destination node
	// 4. Verify transfer
	
	// For now, we'll simulate a transfer with a sleep and random success
	time.Sleep(200 * time.Millisecond)
	success := rand.Float32() > 0.2 // 80% success rate for simulation
	
	if !success {
		task.LastError = "Simulated transfer failure"
	}
	
	return success
}

// GetTaskStatus returns the status of a task
func (ts *TransferService) GetTaskStatus(taskID string) (*TransferTask, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, false
	}
	
	// Return a copy to avoid concurrent access issues
	taskCopy := *task
	return &taskCopy, true
}

// GetAllTasks returns all tasks
func (ts *TransferService) GetAllTasks() []*TransferTask {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	tasks := make([]*TransferTask, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	
	return tasks
}

// CancelTask cancels a running task
func (ts *TransferService) CancelTask(taskID string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	task, exists := ts.tasks[taskID]
	if !exists || task.State == Completed || task.State == Failed {
		return false
	}
	
	ts.logger.Info("Cancelling task %s", taskID)
	
	task.State = Failed
	task.LastError = "Task cancelled by user"
	task.UpdatedAt = time.Now().UnixNano()
	
	// Remove from queue if it's still there
	for i, t := range ts.taskQueue {
		if t.ID == taskID {
			// Remove from queue
			ts.taskQueue = append(ts.taskQueue[:i], ts.taskQueue[i+1:]...)
			break
		}
	}
	
	return true
}

// GetTaskMetrics returns metrics about the transfer service
func (ts *TransferService) GetTaskMetrics() map[string]int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	pending := 0
	inProgress := 0
	completed := 0
	failed := 0
	retrying := 0
	
	for _, task := range ts.tasks {
		switch task.State {
		case Pending:
			pending++
		case InProgress:
			inProgress++
		case Completed:
			completed++
		case Failed:
			failed++
		case Retrying:
			retrying++
		}
	}
	
	return map[string]int{
		"pending":     pending,
		"inProgress":  inProgress,
		"completed":   completed,
		"failed":      failed,
		"retrying":    retrying,
		"queueLength": len(ts.taskQueue),
		"activeTasks": ts.activeTasks,
	}
}