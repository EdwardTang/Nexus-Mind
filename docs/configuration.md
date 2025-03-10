# Nexus-Mind Configuration Guide

This document provides detailed information on configuring your Nexus-Mind cluster.

## Environment Variables

Nexus-Mind can be configured using the following environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `NODE_ID` | Unique identifier for the node | - | Yes |
| `HTTP_PORT` | Port for the HTTP API | 8080 | No |
| `CLUSTER_ADDRESSES` | Comma-separated list of other node addresses | - | No |
| `DIMENSIONS` | Vector dimensions | 128 | No |
| `DISTANCE_FUNCTION` | Similarity metric (cosine, euclidean, dot) | cosine | No |
| `REPLICATION_FACTOR` | Number of replicas for each vector | 1 | No |
| `SEEDS` | Comma-separated list of seed nodes for joining | - | No |
| `DATA_DIR` | Directory to store persistent data | ./data | No |
| `VIRTUAL_NODES` | Number of virtual nodes in the token ring | 256 | No |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info | No |
| `MAX_CONNECTIONS` | Maximum number of concurrent connections | 1000 | No |
| `QUERY_TIMEOUT_MS` | Query timeout in milliseconds | 5000 | No |

## Configuration File

Alternatively, you can use a YAML configuration file. Create a file named `config.yaml` with the following format:

```yaml
node:
  id: "node-1"
  httpPort: 8080
  dataDir: "./data"

cluster:
  seeds: ["node-2:8081", "node-3:8082"]
  replicationFactor: 2
  virtualNodes: 256

vector:
  dimensions: 128
  distanceFunction: "cosine"
  
network:
  maxConnections: 1000
  queryTimeoutMs: 5000
  
logging:
  level: "info"
```

To use a configuration file, set the `CONFIG_FILE` environment variable:

```bash
CONFIG_FILE=./config.yaml ./bin/nexus-mind-vector-store
```

## Advanced Configuration

### Token Ring Configuration

The token ring is responsible for distributing vectors across the cluster. You can fine-tune its behavior:

```yaml
tokenRing:
  hashFunction: "murmur3"  # Hash function for token assignment
  rebalanceThreshold: 0.2  # Trigger rebalance when imbalance exceeds this threshold
  rebalanceBatchSize: 100  # Vectors to transfer in a single batch
```

### Performance Tuning

For high-performance environments, consider these settings:

```yaml
performance:
  indexCacheSize: 10000     # Number of vectors to cache in memory
  queryThreads: 4           # Threads for processing queries
  transferConcurrency: 2    # Concurrent vector transfers during rebalancing
  indexRefreshIntervalMs: 1000  # Milliseconds between index refreshes
```

### Security Configuration

```yaml
security:
  tlsEnabled: true
  certFile: "/path/to/cert.pem"
  keyFile: "/path/to/key.pem"
  clientAuth: true
  trustedCACerts: "/path/to/ca.pem"
```

## Docker Environment

When running with Docker, configure your settings in the `docker-compose.yml` file:

```yaml
services:
  node-1:
    image: nexus-mind
    environment:
      - NODE_ID=node-1
      - HTTP_PORT=8080
      - DIMENSIONS=128
      - DISTANCE_FUNCTION=cosine
      - REPLICATION_FACTOR=2
```

## Example Configurations

### Development Single Node

```yaml
node:
  id: "dev-node"
  httpPort: 8080
  dataDir: "./data"

vector:
  dimensions: 128
  distanceFunction: "cosine"
  
logging:
  level: "debug"
```

### Production Cluster

```yaml
node:
  id: "prod-node-1"
  httpPort: 8080
  dataDir: "/var/lib/nexus-mind/data"

cluster:
  seeds: ["prod-node-2:8080", "prod-node-3:8080"]
  replicationFactor: 3
  virtualNodes: 512

vector:
  dimensions: 1536
  distanceFunction: "cosine"
  
network:
  maxConnections: 5000
  queryTimeoutMs: 2000
  
logging:
  level: "info"
  
performance:
  indexCacheSize: 100000
  queryThreads: 16
  transferConcurrency: 4
```

## Monitoring Configuration

```yaml
monitoring:
  enabled: true
  metricsPort: 9090
  healthCheckPath: "/health"
  exporterType: "prometheus"
```

For more details on performance tuning and advanced configurations, see the [Performance Tuning Guide](./performance_tuning.md).