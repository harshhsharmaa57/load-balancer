# ⚖️ Go Load Balancer

A production-oriented **Layer 7 HTTP load balancer** written in pure Go with zero external dependencies. This project demonstrates backend systems architecture, concurrent programming, reverse-proxy networking, and performance benchmarking — all following idiomatic Go conventions and professional project structure.

[![Go Version](https://img.shields.io/badge/Go-1.26.2-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Project Structure](#project-structure)
- [Architecture Deep Dive](#architecture-deep-dive)
  - [Request Lifecycle](#request-lifecycle)
  - [internal/lb/backend.go — Backend Model](#internallbbackendgo--backend-model)
  - [internal/lb/loadbalancer.go — Round-Robin Selection](#internallbloadbalancergo--round-robin-selection)
  - [internal/lb/healthcheck.go — TCP Health Probe](#internallbhealthcheckgo--tcp-health-probe)
  - [cmd/loadbalancer/main.go — HTTP Server & Entrypoint](#cmdloadbalancermaingo--http-server--entrypoint)
  - [cmd/backend/main.go — Mock Backend Server](#cmdbackendmaingo--mock-backend-server)
  - [cmd/stresstest/main.go — Benchmarking Tool](#cmdstresstestmaingo--benchmarking-tool)
- [Concurrency Model](#concurrency-model)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Stress Testing & Benchmarking](#stress-testing--benchmarking)
- [Configuration Reference](#configuration-reference)
- [Design Decisions & Trade-offs](#design-decisions--trade-offs)
- [What This Project Demonstrates](#what-this-project-demonstrates)
- [Evaluation Guide for Recruiters](#evaluation-guide-for-recruiters)
- [Future Improvements](#future-improvements)

---

## Overview

This project implements a **reverse-proxy load balancer** that distributes incoming HTTP requests across a pool of backend servers using a **round-robin algorithm**. It includes:

- A core load balancer with health-aware routing
- Lightweight mock backend servers for local testing
- A full-featured stress testing / benchmarking CLI tool
- Background health checking via TCP probes on a 5-second ticker

The entire system is built using only the **Go standard library** (`net/http`, `net/http/httputil`, `sync`, `sync/atomic`, `net`, `context`, `flag`) — zero third-party dependencies.

**Module:** `github.com/harshhsharmaa57/load-balancer`  
**Go version:** `1.26.2`

---

## Key Features

| Feature | Details |
|---|---|
| **Round-Robin Routing** | Atomic counter-based backend selection across healthy servers |
| **Reverse Proxy** | Built on `net/http/httputil.ReverseProxy` for transparent request forwarding |
| **Health Checking** | Background goroutine performs TCP dial probes every 5 seconds |
| **Thread Safety** | `sync.RWMutex` for health state, `sync/atomic` for the round-robin counter |
| **Server Timeouts** | Configurable `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` on the HTTP server |
| **Request Logging** | Per-request structured logging with method, path, target backend, and latency |
| **Stress Testing** | Built-in CLI tool with configurable concurrency, request count, and timeout |
| **A/B Comparison** | Compare single-server vs. load-balanced performance side by side |
| **Zero Dependencies** | Entire project uses only the Go standard library |

---

## Project Structure

```
load-balancer/
├── go.mod                          # Go module definition (no external deps)
├── README.md                       # This file
├── cmd/                            # Application entrypoints (executables)
│   ├── loadbalancer/
│   │   └── main.go                 # Load balancer HTTP server (47 lines)
│   ├── backend/
│   │   └── main.go                 # Mock backend HTTP server (20 lines)
│   └── stresstest/
│       └── main.go                 # Stress test / benchmark CLI (234 lines)
└── internal/                       # Private packages (not importable externally)
    └── lb/
        ├── backend.go              # Backend struct, reverse proxy, health state (47 lines)
        ├── loadbalancer.go         # Round-robin selection, health check scheduler (36 lines)
        └── healthcheck.go          # TCP health probe function (17 lines)
```

### Why `internal/`?

The `internal/` directory is a Go convention that **prevents external packages from importing** the code inside it. This enforces encapsulation — the `lb` package is only usable by code within this module's `cmd/` packages, keeping the API surface private and the implementation details hidden.

### Why `cmd/`?

Each subdirectory under `cmd/` is a standalone executable (`package main`). This is the standard Go project layout for repositories that produce multiple binaries from a single module.

---

## Architecture Deep Dive

### Request Lifecycle

```
Client Request
      │
      ▼
┌──────────────┐
│  HTTP Server │  :8080  (cmd/loadbalancer)
│  HandleFunc  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ NextBackend()│  Round-robin selection (internal/lb)
│  atomic.Add  │  Skips unhealthy backends
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ ReverseProxy │  httputil.ReverseProxy.ServeHTTP()
│  .ServeHTTP  │  Transparently forwards req/resp
└──────┬───────┘
       │
       ├─────────────┬─────────────┐
       ▼             ▼             ▼
  ┌─────────┐  ┌─────────┐  ┌─────────┐
  │ :9001   │  │ :9002   │  │ :9003   │
  │ Backend │  │ Backend │  │ Backend │
  └─────────┘  └─────────┘  └─────────┘

Background Health Check (every 5s):
  TCP dial to each backend → SetAlive(true/false)
```

---

### `internal/lb/backend.go` — Backend Model

This file defines the **`Backend` struct** — the core data model representing a single upstream server.

#### Struct Definition

```go
type Backend struct {
    URL   *url.URL                   // Parsed URL of the backend (e.g. http://localhost:9001)
    Proxy *httputil.ReverseProxy     // Pre-configured reverse proxy instance
    Alive bool                       // Current health status
    mu    sync.RWMutex               // Protects Alive field for concurrent access
}
```

**Fields explained:**

| Field | Type | Visibility | Purpose |
|---|---|---|---|
| `URL` | `*url.URL` | Exported | Parsed backend address used for health checks and logging |
| `Proxy` | `*httputil.ReverseProxy` | Exported | Pre-built reverse proxy; calls `ServeHTTP()` to forward requests |
| `Alive` | `bool` | Exported | Health flag toggled by the background health checker |
| `mu` | `sync.RWMutex` | Unexported | Ensures thread-safe reads/writes to `Alive` from multiple goroutines |

#### Methods

**`SetAlive(alive bool)`** — Write-locks `mu`, sets `Alive`, unlocks. Called by the health checker goroutine.

**`isAlive() bool`** — Read-locks `mu`, returns `Alive`, defers unlock. Called by `NextBackend()` during request routing. Using `RLock` allows multiple concurrent readers without blocking each other.

#### `NewBackends()` — Factory Function

Creates the backend pool from a hardcoded list of URLs (`localhost:9001`, `9002`, `9003`). For each URL:
1. Parses the raw string into a `*url.URL`
2. Creates an `httputil.NewSingleHostReverseProxy(u)` — this configures the proxy's `Director` function to rewrite incoming request URLs to target the backend
3. Appends a `*Backend` to the slice

Returns `[]*Backend` — a slice of pointers, ensuring all references point to the same struct instances.

---

### `internal/lb/loadbalancer.go` — Round-Robin Selection

#### Global Counter

```go
var counter uint64
```

A **package-level atomic counter** that persists across all calls to `NextBackend()`. This is the backbone of the round-robin algorithm — each call atomically increments it and uses modulo to pick the next backend.

#### `NextBackend(backends []*Backend) *Backend`

The core routing function:

```go
func NextBackend(backends []*Backend) *Backend {
    n := uint64(len(backends))
    for i := uint64(0); i < n; i++ {
        next := atomic.AddUint64(&counter, 1)  // Atomically increment
        b := backends[next%n]                   // Modulo for round-robin

        if b.isAlive() {
            return b           // Return first healthy backend found
        }
        n = n - uint64(1)     // Narrow search window (note: see trade-offs)
    }
    return nil                 // All backends unhealthy
}
```

**Step-by-step flow:**
1. Get the total number of backends (`n`)
2. Loop up to `n` times (worst case: all but one backend is down)
3. Atomically increment the global counter — this is **lock-free** and safe for concurrent goroutines
4. Compute `next % n` to select a backend index
5. If that backend is alive, return it
6. If not, decrement `n` and try again
7. If all backends are unhealthy, return `nil`

**Why `sync/atomic` instead of a mutex?** Atomic operations are significantly faster than mutex lock/unlock cycles for a simple counter — critical when handling thousands of concurrent requests.

#### `StartHealthCheck(backends []*Backend)`

Starts a **background goroutine** that runs health checks on a fixed interval:

```go
func StartHealthCheck(backends []*Backend) {
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        for range ticker.C {
            for _, b := range backends {
                alive := isBackendAlive(b.URL)
                b.SetAlive(alive)
            }
        }
    }()
}
```

- Creates a `time.Ticker` that fires every **5 seconds**
- Spawns a goroutine that iterates all backends on each tick
- Calls `isBackendAlive()` (TCP probe) and updates health state via `SetAlive()`
- The goroutine runs for the lifetime of the process (no stop mechanism — intentional simplicity)

---

### `internal/lb/healthcheck.go` — TCP Health Probe

```go
func isBackendAlive(u *url.URL) bool {
    conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}
```

**How it works:**
1. Attempts a **TCP connection** to the backend's `host:port`
2. Uses a **2-second timeout** — if the backend doesn't respond within 2s, it's considered down
3. If the connection succeeds, immediately closes it and returns `true`
4. If the dial fails (connection refused, timeout, DNS error), returns `false`

**Why TCP instead of HTTP?** A TCP dial is the lightest possible check — it verifies the backend process is listening on the port without sending an HTTP request. This minimizes overhead on both the load balancer and the backends.

---

### `cmd/loadbalancer/main.go` — HTTP Server & Entrypoint

The main load balancer application (47 lines):

```go
func main() {
    backends := lb.NewBackends()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        b := lb.NextBackend(backends)

        if b == nil {
            http.Error(w, "no healthy backends", http.StatusServiceUnavailable)
            return
        }

        b.Proxy.ServeHTTP(w, r)

        log.Printf("[%s] %s %s  →  %s  (%v)",
            time.Now().Format("15:04:05"),
            r.Method, r.URL.Path, b.URL.Host,
            time.Since(start),
        )
    })

    lb.StartHealthCheck(backends)

    srv := &http.Server{
        Addr:         ":8080",
        Handler:      http.DefaultServeMux,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    log.Println("LB starting on :8080")
    log.Fatal(srv.ListenAndServe())
}
```

**Execution flow:**
1. **Initialize backends** — calls `lb.NewBackends()` to create the pool of 3 backend instances
2. **Register handler** — a catch-all `"/"` handler that routes every incoming request
3. **Request handling** — for each request:
   - Records start time for latency measurement
   - Calls `NextBackend()` to get the next healthy backend
   - Returns `503 Service Unavailable` if all backends are down
   - Forwards the request via `ReverseProxy.ServeHTTP()`
   - Logs the request with timestamp, method, path, target, and duration
4. **Start health checks** — launches the background health checker
5. **Start server** — creates an `http.Server` with explicit timeouts and listens on `:8080`

**Server Timeout Configuration:**

| Timeout | Value | Purpose |
|---|---|---|
| `ReadTimeout` | 5s | Max time to read the entire request (headers + body) |
| `WriteTimeout` | 10s | Max time to write the response |
| `IdleTimeout` | 60s | Max time a keep-alive connection can remain idle |

These timeouts protect against **slowloris attacks** and **resource exhaustion** from hanging connections.

---

### `cmd/backend/main.go` — Mock Backend Server

A minimal HTTP server (20 lines) used for testing:

```go
func main() {
    port := os.Args[1]
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Response from backend %s", port)
    })
    log.Printf("Backend is running at port %s", port)
    log.Fatal(http.ListenAndServe(port, nil))
}
```

- Takes the port as a **command-line argument** (`os.Args[1]`)
- Responds to all requests with `"Response from backend :XXXX"`
- Used to verify round-robin distribution and health check behavior

---

### `cmd/stresstest/main.go` — Benchmarking Tool

A fully-featured stress testing CLI (234 lines) for measuring load balancer performance.

#### `Stats` Struct

```go
type Stats struct {
    TotalRequests      uint64
    SuccessfulRequests uint64
    FailedRequests     uint64
    TotalBytes         uint64
    TotalLatency       int64
    MinLatency         int64
    MaxLatency         int64
    StatusCodes        map[int]uint64
    mu                 sync.Mutex
}
```

Tracks all benchmark metrics using a **hybrid concurrency model**:
- **Atomic operations** (`sync/atomic`) for counters that are incremented frequently (`TotalRequests`, `SuccessfulRequests`, `FailedRequests`, `TotalBytes`, `TotalLatency`)
- **Mutex** (`sync.Mutex`) for the `StatusCodes` map and min/max latency comparisons (which require read-modify-write and can't be done atomically)

#### `stressTest()` Function

```go
func stressTest(targetURL string, concurrency int, totalRequests int, timeout time.Duration) (*Stats, time.Duration)
```

**Concurrency control:**
- Uses a **semaphore pattern** via a buffered channel: `semaphore := make(chan struct{}, concurrency)`
- Each goroutine acquires a semaphore slot before executing and releases it on completion
- A `sync.WaitGroup` tracks completion of all request goroutines

**HTTP client configuration:**
```go
client := &http.Client{
    Timeout: timeout,
    Transport: &http.Transport{
        MaxIdleConns:        concurrency * 2,
        MaxIdleConnsPerHost: concurrency * 2,
        IdleConnTimeout:     30 * time.Second,
        DisableKeepAlives:   false,
    },
}
```

| Setting | Value | Purpose |
|---|---|---|
| `Timeout` | User-configurable (default 10s) | Per-request timeout |
| `MaxIdleConns` | `concurrency × 2` | Total idle connection pool size |
| `MaxIdleConnsPerHost` | `concurrency × 2` | Idle connections per backend host |
| `IdleConnTimeout` | 30s | How long idle connections live |
| `DisableKeepAlives` | `false` | Reuse TCP connections (HTTP keep-alive) |

**Per-request flow:**
1. Create a context with timeout
2. Build an HTTP GET request
3. Execute the request, measure latency
4. Read and discard the response body
5. Record success (2xx) or failure

#### Comparison Mode (`-compare`)

When `--compare` is passed, the tool runs **two sequential stress tests**:
1. First against the single server URL (`-single`)
2. Then against the load balancer URL (`-lb`)

It then prints a side-by-side comparison table with:
- Requests/second
- Average latency
- Success rate
- Total failures
- Percentage change between the two

---

## Concurrency Model

The project uses three distinct concurrency primitives:

| Primitive | Location | Purpose |
|---|---|---|
| `sync/atomic.AddUint64` | `loadbalancer.go` | Lock-free round-robin counter increment |
| `sync.RWMutex` | `backend.go` | Readers-writer lock for backend health state |
| `sync.Mutex` | `stresstest/main.go` | Protects status code map and min/max latency |
| `sync.WaitGroup` | `stresstest/main.go` | Waits for all benchmark goroutines to finish |
| Buffered channel (semaphore) | `stresstest/main.go` | Limits concurrent in-flight requests |
| `time.Ticker` + goroutine | `loadbalancer.go` | Periodic background health checks |
| `context.WithTimeout` | `stresstest/main.go` | Per-request cancellation and deadline |

---

## Prerequisites

- **Go 1.26.2+** installed and available in `PATH`
- No external dependencies required — the project uses only the Go standard library

---

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/harshhsharmaa57/load-balancer.git
cd load-balancer
```

### 2. Start Backend Servers

Open **three separate terminals** and start one backend instance in each:

```powershell
# Terminal 1
go run ./cmd/backend :9001

# Terminal 2
go run ./cmd/backend :9002

# Terminal 3
go run ./cmd/backend :9003
```

Each backend will log: `Backend is running at port :900X`

### 3. Start the Load Balancer

In a **fourth terminal**:

```powershell
go run ./cmd/loadbalancer
```

Output: `LB starting on :8080`

### 4. Send Test Requests

```powershell
# Single request
curl http://localhost:8080/

# Multiple requests to observe round-robin
for ($i=0; $i -lt 6; $i++) { curl -s http://localhost:8080/; Write-Host "" }
```

Expected output (rotating across backends):
```
Response from backend :9001
Response from backend :9002
Response from backend :9003
Response from backend :9001
Response from backend :9002
Response from backend :9003
```

### 5. Test Health Check Behavior

1. Stop one backend (e.g., kill the `:9002` terminal)
2. Wait ~5 seconds for the health check to detect the failure
3. Send requests — they will only route to `:9001` and `:9003`
4. Restart the backend — it will rejoin the pool after the next health check

---

## Stress Testing & Benchmarking

### Basic Stress Test

```powershell
go run ./cmd/stresstest -target http://localhost:8080 -c 100 -n 10000
```

### Comparison Mode

```powershell
go run ./cmd/stresstest -compare -single http://localhost:9001 -lb http://localhost:8080 -c 100 -n 10000
```

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `-target` | `http://localhost:8080` | URL to stress test |
| `-c` | `100` | Number of concurrent connections |
| `-n` | `10000` | Total number of requests to send |
| `-t` | `10s` | Per-request timeout (Go duration format) |
| `-compare` | `false` | Enable A/B comparison mode |
| `-single` | `http://localhost:9001` | Single server URL (comparison mode) |
| `-lb` | `http://localhost:8080` | Load balancer URL (comparison mode) |

### Sample Output

```
============================================================
  📊 STRESS TEST RESULTS
============================================================
  ⏱️  Duration:           2.45s
  📨 Total Requests:     10000
  ✅ Successful:         10000 (100.00%)
  ❌ Failed:             0 (0.00%)
  🚀 Requests/Second:    4081.63
  📦 Total Data:         263.67 KB
  📥 Throughput:         107.62 KB/s
  ⏳ Avg Latency:        24.123ms
  ⚡ Min Latency:        512.1µs
  🐢 Max Latency:        198.234ms

  📋 Status Code Distribution:
     200 (OK): 10000
============================================================
```

---

## Configuration Reference

### Load Balancer (`cmd/loadbalancer`)

| Parameter | Value | Configurable | Location |
|---|---|---|---|
| Listen address | `:8080` | Hardcoded | `cmd/loadbalancer/main.go:37` |
| Backend URLs | `:9001`, `:9002`, `:9003` | Hardcoded | `internal/lb/backend.go:29-33` |
| Health check interval | `5s` | Hardcoded | `internal/lb/loadbalancer.go:25` |
| Health check timeout (TCP) | `2s` | Hardcoded | `internal/lb/healthcheck.go:10` |
| HTTP ReadTimeout | `5s` | Hardcoded | `cmd/loadbalancer/main.go:39` |
| HTTP WriteTimeout | `10s` | Hardcoded | `cmd/loadbalancer/main.go:40` |
| HTTP IdleTimeout | `60s` | Hardcoded | `cmd/loadbalancer/main.go:41` |

### Backend Server (`cmd/backend`)

| Parameter | Value | Configurable |
|---|---|---|
| Listen port | CLI argument (`os.Args[1]`) | ✅ Yes |

---

## Design Decisions & Trade-offs

### Round-Robin vs. Other Algorithms

**Chosen:** Simple round-robin with atomic counter.  
**Why:** Predictable distribution, minimal overhead, easy to reason about. For a demonstration project, it clearly shows the load balancing concept without over-engineering.  
**Trade-off:** Does not account for backend load, response times, or connection counts. A production system might use least-connections or weighted round-robin.

### TCP Health Check vs. HTTP Health Check

**Chosen:** Raw TCP dial (`net.DialTimeout`).  
**Why:** Minimal overhead — just verifies the process is listening. No HTTP request parsing on the backend side.  
**Trade-off:** Cannot detect application-level failures (e.g., a backend returning 500s but still accepting TCP connections). A production system would add an HTTP `/health` endpoint.

### Hardcoded Configuration vs. Environment/CLI Flags

**Chosen:** Backend URLs and ports are hardcoded.  
**Why:** Keeps the code simple and focused on the core concepts. Configuration management is orthogonal to load balancing.  
**Trade-off:** Requires code changes to add/remove backends.

### Package-Level Atomic Counter vs. Struct Field

**Chosen:** `var counter uint64` at package level.  
**Why:** Simplifies the API — `NextBackend()` is a standalone function, no need to instantiate a load balancer object.  
**Trade-off:** Only one load balancer instance per process. A struct-based approach would allow multiple independent balancers.

---

## What This Project Demonstrates

| Skill | Evidence |
|---|---|
| **Go concurrency** | Goroutines, `sync.RWMutex`, `sync/atomic`, `sync.WaitGroup`, channels as semaphores |
| **Networking** | TCP connections, HTTP servers, reverse proxies, keep-alive tuning |
| **Standard library mastery** | `net/http`, `net/http/httputil`, `net`, `context`, `flag`, `time`, `io` |
| **Project structure** | `cmd/` + `internal/` layout following Go conventions |
| **Systems design** | Health checking, failover, request distribution, timeout configuration |
| **Benchmarking** | Custom stress test tool with statistical reporting |
| **Code quality** | Small focused files, clear naming, minimal dependencies |

---

## Evaluation Guide for Recruiters

If reviewing this repository, focus on these aspects:

1. **Architecture** — `cmd/` for entrypoints, `internal/` for encapsulation. Clean separation of concerns.
2. **Concurrency correctness** — `sync.RWMutex` for shared health state, `sync/atomic` for the lock-free counter, semaphore pattern for concurrency limiting.
3. **Networking** — Reverse proxy forwarding, TCP health probes, HTTP server timeouts for resilience.
4. **Code clarity** — Each file has a single responsibility. The entire core logic is ~100 lines. No unnecessary abstractions.
5. **Operational awareness** — Request logging with latency, health check intervals, timeout configuration, graceful 503 responses when all backends are down.

---

## Build & Verify

```powershell
# Verify all packages compile
go build ./...

# Run tests (if any)
go test ./...

# Build individual binaries
go build -o lb.exe ./cmd/loadbalancer
go build -o backend.exe ./cmd/backend
go build -o stresstest.exe ./cmd/stresstest
```

---

## Future Improvements

- [ ] CLI flags or environment variables for backend URLs and listen port
- [ ] HTTP-based health checks (`GET /health`) for application-level validation
- [ ] Weighted round-robin or least-connections algorithm
- [ ] Graceful shutdown with `os.Signal` handling
- [ ] Prometheus metrics endpoint for observability
- [ ] Rate limiting and circuit breaker patterns
- [ ] TLS termination support
- [ ] Dynamic backend registration via an admin API
- [ ] Configurable health check interval and timeout
- [ ] Request retry on backend failure

---

## Commit History

| Date | Commit | Description |
|---|---|---|
| 2026-06-11 | `43ea8e0` | Initial project scaffold |
| 2026-06-12 | `f3c8add` | Core load balancer round-robin logic |
| 2026-06-13 | `9dd93ef` | Health checking and backend filtering |
| 2026-06-14 | `4b426a2` | HTTP server timeout configuration |
| 2026-06-14 | `bb91dfd` | Per-request logging with latency |
| 2026-06-20 | `731cb33` | Project structure and README |

---

## License

This repository is intended for **technical evaluation and portfolio demonstration**. It showcases backend engineering fundamentals in Go rather than serving as a production-grade load balancer.

---

> Built with ❤️ in Go — zero external dependencies, pure standard library.
