
# Nexus‑Mind

> **TL;DR** – Nexus‑Mind is a *teaching* vector database: tiny, readable, and eager for contributions that push it from “toy” to “tool”.


*A tiny, hack‑friendly vector store that piggy‑backs on an educational Go Raft implementation. Perfect for experiments, **not** production (yet!).*

---

## Why another vector DB?
Nexus‑Mind started as a playground for learning:
* **Consensus** – borrow the Raft labs code from MIT 6.824 for strong consistency.  
* **Vector search internals** – write the simplest possible index first (`LinearIndex`), then iterate.
* **Control‑plane exploration** – prototype how Raft (strong, synchronous) and Gossip (weak, asynchronous) can coexist without stepping on each other.

If you just need a battle‑tested store, use Milvus, Qdrant or Weaviate.  
If you want to tweak every line and watch a DB grow, read on.

---

## Features (April 2025)

| Area | Status | Notes |
|------|--------|-------|
| **Vector CRUD & K‑NN** | ✅ Works (linear scan) | See `src/vector/index/linear.go` |
| **HTTP API** | ✅ Basic `/collections/:name/query` | JSON; no auth |
| **In‑memory persistence** | ✅ | Vectors lost on restart |
| **Raft log replication** | 🛠 Library is there, **not wired into vector store** | From MIT 6.824 labs |
| **Shard & KV layers** | 🛠 Present but demo‑only | `src/shard*` |
| **Gossip membership** | 🚧 Planned | Design in `docs/ARCHITECTURE.md` |
| **Advanced indexes (HNSW/IVF/PQ)** | 🚧 | Road‑mapped |
| **Docker compose cluster** | 🚧 | `./run.sh` is a placeholder |

---

## Quick start (single node)

```bash
# Prereqs: Go 1.21+
git clone https://github.com/EdwardTang/Nexus-Mind.git
cd Nexus-Mind/src
go run ./main.go
```

Open another terminal and try:

```bash
# Add a vector
curl -X POST localhost:8080/vectors \
     -H "Content-Type: application/json" \
     -d '{"id":"vec1","vector":[0.1,0.2,0.3]}'

# Similarity search (top‑5)
curl -X POST localhost:8080/search \
     -H "Content-Type: application/json" \
     -d '{"vector":[0.1,0.2,0.3],"k":5}'
```

---

## Architecture snapshot

```
Client ──HTTP/JSON──▶ Query Router
                         │
                         ▼
                 +--------------+
                 | Collection   |
                 |  (in‑mem)    |
                 +--------------+
                         │
                   +-----------+
                   | Index     |  ← Linear scan today
                   +-----------+
```

*Future*: multiple collections share a **token‑ring** sharding layer; each shard is a Raft group for strong writes. Gossip/ SWIM spreads node liveness and ring changes.

Detailed design docs live in [`/docs`](docs):

* `ARCHITECTURE.md` – 3‑layer blueprint (coordination, storage, query).  
* `vector_store_layer.md` – ideas for HNSW & disk persistence.  
* `ROADMAP.md` – milestone tracker.

---

## Code layout

```text
src/
├── raft/          – Stand‑alone Raft library (leader election, log, snapshot)
├── vector/
│   ├── index/     – LinearIndex (baseline)
│   ├── query/     – HTTP layer & filter DSL
│   └── distance.go – SIMD‑backed metrics
├── shard*         – Experiments with sharded KV on top of Raft
└── main.go        – Demo server wiring everything together
docs/ …            – design & progress notes
```

---

## Roadmap highlights

1. **Wire Raft into vector mutations** – WAL + replicas.
2. **Replace linear scan with HNSW** – keep brute‑force as fall‑back.
3. **Cluster bootstrap & gossip membership** – SWIM‑like heartbeat.
4. **On‑disk segments** – mmap + background compaction.
5. **Observability** – Prometheus metrics & Jaeger traces.

See [`docs/ROADMAP.md`](docs/ROADMAP.md) for the full list.

---

## Contributing

PRs and design sketches are welcome! Start with a good first issue:

* `vector/index`: plug in any ANN algorithm you fancy.
* `raft`: add snapshotting tests.
* Docs proofreading.

Please run `go test ./...` before opening a PR.

---

## License

Apache 2.0 – see `LICENSE`.
