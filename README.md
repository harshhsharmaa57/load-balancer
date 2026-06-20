# Go Load Balancer

A production-oriented Go load balancer implementation designed to demonstrate backend architecture, maintainable package structure, and performance testing for distributed services.

## Overview

This project implements a simple yet practical load balancer in Go, with a focus on clean separation of concerns and extensibility:

- `cmd/loadbalancer/`: load balancer entrypoint and HTTP server configuration
- `cmd/backend/`: lightweight backend server for functional testing and demonstration
- `cmd/stresstest/`: stress test harness for benchmarking load balancer performance
- `internal/lb/`: private load balancer package containing routing, backend modeling, and health checking

The codebase is organized to reflect professional backend engineering practices, separating executable applications from internal business logic.

## What this project demonstrates

- Core Go backend skills: concurrency, networking, synchronization, and package organization
- Reverse proxy usage with `net/http/httputil`
- Thread-safe backend health state management using `sync.RWMutex`
- Load balancing using a simple round-robin algorithm with atomic counters
- Periodic health checks implemented with a background ticker
- Real-world operational concerns: timeouts, logging, and graceful routing behavior
- Benchmarking and comparison of a load balanced cluster vs. a single server

## Architecture

### `internal/lb`

This package encapsulates the load balancer implementation and is intentionally kept private to the module.

- `backend.go`
  - `Backend` type holds backend metadata, reverse proxy instance, and health status
  - thread-safe accessors ensure safe concurrent reads and writes
- `loadbalancer.go`
  - round-robin backend selection using `sync/atomic`
  - background health checker startup function
- `healthcheck.go`
  - lightweight TCP health probe to detect responsive backends

### `cmd/loadbalancer`

The load balancer service exposes a single HTTP endpoint and forwards incoming requests to healthy backends. It also logs request handling latency and backend target information.

### `cmd/backend`

A simple backend HTTP server that responds with a port-specific message. This service is used to verify load balancer routing and health-check behavior in a local environment.

### `cmd/stresstest`

A dedicated tool for measuring load balancer performance. It supports:

- configurable concurrency and request count
- request timeout handling
- success/failure aggregation
- latency, throughput, and status distribution reporting
- optional comparison between a single backend and the load balancer

## How to run

### Start backend servers

Open three terminals and start the backend instances:

```powershell
cd load-balancer/cmd/backend
go run main.go :9001
```

```powershell
cd load-balancer/cmd/backend
go run main.go :9002
```

```powershell
cd load-balancer/cmd/backend
go run main.go :9003
```

### Start the load balancer

```powershell
cd load-balancer/cmd/loadbalancer
go run main.go
```

The load balancer listens on `http://localhost:8080` and forwards traffic to healthy backends.

### Verify behavior

Send a request through the load balancer:

```powershell
curl http://localhost:8080/
```

You should receive a response from one of the backend instances.

## Stress test and benchmark

Run a stress test against the load balancer:

```powershell
cd load-balancer/cmd/stresstest
go run main.go -target http://localhost:8080 -c 100 -n 10000
```

Compare the load balancer against a single backend server:

```powershell
go run main.go -compare -single http://localhost:9001 -lb http://localhost:8080 -c 100 -n 10000
```

## Professional highlights

- Clean package structure suitable for real backend services
- Private internal package exposure pattern using `internal/`
- Separation of concerns between command entrypoints and business logic
- Explicit health-check strategy for backend reliability
- Benchmarking support for performance validation
- Lightweight and idiomatic Go code

## Evaluation guide for recruiters

If you are reviewing this repository, focus on these aspects:

1. **Architecture and folder layout**
   - `cmd/` for application entrypoints
   - `internal/` for package encapsulation
2. **Concurrency correctness**
   - use of `sync.RWMutex` for backend health state
   - use of `sync/atomic` for round-robin counters
3. **Networking and resilience**
   - reverse proxy forwarding
   - TCP-based health checks
   - request timeout handling
4. **Code clarity and maintainability**
   - straightforward package boundaries
   - concise and readable logic
   - minimal external dependencies

## Build and test

Verify the module and package build with:

```powershell
go test ./...
```

This will confirm that the load balancer, backend, and stress-test packages compile successfully.

## Notes

This repository is intentionally designed to showcase backend engineering fundamentals in Go, rather than a full production-grade load balancer.
It is appropriate for technical evaluation and demonstrates ability in architecture, concurrency, and service design.
