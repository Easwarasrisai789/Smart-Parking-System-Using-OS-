# Performance and profiling

## Environment

Record the OS, CPU, memory, Go version, and whether the run was performed on the host or inside the Dev Container. The numbers below must come from an actual run; do not use them as cross-machine comparisons.

## Commands

Run LSM benchmarks:

```sh
go test -bench=. -benchmem ./lsm
```

Run the concurrent parking gRPC stress client while `cmd/server` is running:

```sh
go run ./cmd/stress -address localhost:50051 -clients 100 -operations 1000
```

The stress command reports successful and failed RPCs, duration, successful requests per second, and mean request latency. Each operation attempts an ENTER and, if ENTER succeeds, an EXIT. This currently benchmarks the standalone parking server, not the Kubernetes Raft cluster.

On Linux/WSL2/Dev Container, useful environment commands are:

```sh
uname -a
go version
lscpu
free -h
```

## Results

| Workload | Environment | Result |
|---|---|---|
| LSM benchmarks | Pending measured run | Not recorded |
| Parking gRPC stress | Pending measured run | Not recorded |

No comparisons with RocksDB, etcd, or TiKV have been performed.
