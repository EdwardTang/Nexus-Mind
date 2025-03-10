#!/bin/bash

# Simple management script for tiny-raft vector store

set -e

# Define colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

usage() {
  echo -e "${YELLOW}Nexus-Mind Vector Store Management Script${NC}"
  echo ""
  echo "Usage: $0 [command]"
  echo ""
  echo "Commands:"
  echo "  build          Build Docker images"
  echo "  start          Start the cluster (all nodes)"
  echo "  start-single   Start a single node"
  echo "  stop           Stop the cluster"
  echo "  logs           Show logs for all nodes"
  echo "  logs [node]    Show logs for a specific node (e.g., node-1)"
  echo "  test           Run tests"
  echo "  clean          Remove Docker containers and volumes"
  echo "  status         Show status of running nodes"
  echo "  help           Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0 build       # Build the Docker images"
  echo "  $0 start       # Start the full cluster"
  echo "  $0 logs node-1 # Show logs for node-1"
  echo ""
}

build() {
  echo -e "${GREEN}Building Docker images...${NC}"
  docker-compose build
}

start_cluster() {
  echo -e "${GREEN}Starting the cluster...${NC}"
  docker-compose up -d
  echo -e "${GREEN}Cluster started. Access the API at:${NC}"
  echo "  - Node 1: http://localhost:8080"
  echo "  - Node 2: http://localhost:8081"
  echo "  - Node 3: http://localhost:8082"
}

start_single() {
  echo -e "${GREEN}Starting a single node...${NC}"
  docker-compose up -d node-1
  echo -e "${GREEN}Node started. Access the API at:${NC}"
  echo "  - http://localhost:8080"
}

stop_cluster() {
  echo -e "${GREEN}Stopping the cluster...${NC}"
  docker-compose down
}

show_logs() {
  if [ -z "$1" ]; then
    echo -e "${GREEN}Showing logs for all nodes...${NC}"
    docker-compose logs -f
  else
    echo -e "${GREEN}Showing logs for $1...${NC}"
    docker-compose logs -f "$1"
  fi
}

run_tests() {
  echo -e "${GREEN}Running tests...${NC}"
  docker-compose run --rm node-1 sh -c "cd /app && go test -v ./vectorstore"
}

clean() {
  echo -e "${GREEN}Cleaning up Docker containers and volumes...${NC}"
  docker-compose down -v
  echo -e "${GREEN}Cleanup complete.${NC}"
}

status() {
  echo -e "${GREEN}Node status:${NC}"
  docker-compose ps
}

# Main logic
case "$1" in
  build)
    build
    ;;
  start)
    start_cluster
    ;;
  start-single)
    start_single
    ;;
  stop)
    stop_cluster
    ;;
  logs)
    show_logs "$2"
    ;;
  test)
    run_tests
    ;;
  clean)
    clean
    ;;
  status)
    status
    ;;
  help|"")
    usage
    ;;
  *)
    echo -e "${RED}Unknown command: $1${NC}"
    usage
    exit 1
    ;;
esac

exit 0