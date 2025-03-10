#!/bin/sh
set -e

# Convert environment variables to command line arguments
ARGS=""

# Add mandatory parameters
if [ ! -z "$NODE_ID" ]; then
  ARGS="$ARGS --node-id $NODE_ID"
fi

if [ ! -z "$HTTP_PORT" ]; then
  ARGS="$ARGS --http-port $HTTP_PORT"
fi

if [ ! -z "$DIMENSIONS" ]; then
  ARGS="$ARGS --dimensions $DIMENSIONS"
fi

if [ ! -z "$DISTANCE_FUNCTION" ]; then
  ARGS="$ARGS --distance-function $DISTANCE_FUNCTION"
fi

if [ ! -z "$LOG_LEVEL" ]; then
  ARGS="$ARGS --log-level $LOG_LEVEL"
fi

# Add optional cluster nodes parameter
if [ ! -z "$CLUSTER_NODES" ]; then
  ARGS="$ARGS --cluster-nodes $CLUSTER_NODES"
fi

# Add data directory
ARGS="$ARGS --data-dir /home/appuser/data"

echo "Starting nexus-mind vector store with arguments: $ARGS"
exec nexus-mind-vector-store $ARGS