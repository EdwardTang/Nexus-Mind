# Vector Store Architecture (MVP)

## System Overview

The system is a distributed vector store built on Raft consensus with three primary layers:

1. **Coordination Layer**: Uses Raft consensus for metadata management
2. **Storage Layer**: Manages vector data and document storage
3. **Query Layer**: Handles request processing and similarity search

## Component Architecture

### Coordination Layer

```
┌────────────────────────┐
│   Cluster Coordinator  │
├────────────────────────┤
│ - Raft State Machine   │
│ - Membership Management│
│ - Topology Metadata    │
└────────────────────────┘
```

- **Raft State Machine**: Extended from Nexus-Mind implementation
  - Handles cluster consensus on metadata
  - Manages node state transitions
  - Implements log replication for configuration changes

- **Membership Service**:
  - Node registration/deregistration
  - Health monitoring (heartbeat mechanism)
  - Failure detection and recovery coordination

- **Metadata Manager**:
  - Partition mapping (consistent hashing)
  - Replication strategy configuration
  - Token ring management

### Storage Layer

```
┌────────────────────────┐  ┌────────────────────────┐
│    Vector Storage      │  │   Document Storage     │
├────────────────────────┤  ├────────────────────────┤
│ - Vector Index (HNSW)  │  │ - KV Document Store    │
│ - Raw Vector Data      │  │ - Metadata Index       │
└────────────────────────┘  └────────────────────────┘
```

- **Vector Store**:
  - HNSW Index implementation (optimized for 1536d vectors)
  - Vector persistence to disk
  - In-memory search index for performance

- **Document Store** (optional):
  - Simple key-value storage for document chunks
  - Linked to vector embeddings via IDs
  - Basic metadata indexing

### Query Layer

```
┌────────────────────────┐  ┌────────────────────────┐
│     Query Router       │  │   Similarity Engine    │
├────────────────────────┤  ├────────────────────────┤
│ - Request Distribution │  │ - Distance Calculations│
│ - Result Aggregation   │  │ - KNN Implementation   │
└────────────────────────┘  └────────────────────────┘
```

- **Query Router**:
  - Distributes queries based on vector partitioning
  - Implements quorum-based read/write consistency
  - Aggregates results from multiple nodes

- **Similarity Engine**:
  - Implements vector distance metrics (cosine, dot, euclidean)
  - K-nearest neighbor algorithm
  - Result scoring and ranking

### Client Interface

```
┌────────────────────────┐
│       REST API         │
├────────────────────────┤
│ - CRUD Operations      │
│ - Search Endpoints     │
│ - Batch Processing     │
└────────────────────────┘
```

- **API Service**:
  - RESTful endpoints for vector operations
  - Authentication middleware
  - Bulk import/export handlers

## Data Flow

1. **Write Operation**:
   ```
   Client → API → Query Router → [Calculate Partition] → 
   Write to N replicas → Wait for W acknowledgements → Return success
   ```

2. **Read/Search Operation**:
   ```
   Client → API → Query Router → [Calculate Partitions] → 
   Query M replicas → Wait for R results → Aggregate → Return results
   ```

## Persistence Strategy

- **Vector Data**: Custom binary format optimized for vector storage
- **Index Structure**: HNSW graph serialized to disk for persistence
- **Document Storage**: Simple key-value format with JSON metadata

## Implementation Considerations

- **Language/Framework**: Go (leveraging the existing tiny-raft codebase)
- **Node Communication**: gRPC for efficient binary communication
- **Concurrency Model**: Goroutines with carefully managed shared state
- **Data Locality**: Optimized data placement to minimize network hops

## MVP Limitations

- Manual cluster scaling (no auto-scaling)
- No automatic data rebalancing on node changes
- Single index algorithm choice (HNSW only)
- Basic authentication (API keys only)
- No encryption for data at rest or in transit