# GoSim-Iter 🚀
### Discrete Event Simulation Engine for Go 1.23+

[🇧🇷 Leia em Português](readme.md)

**GoSim-Iter** is a high-performance discrete event simulation (DES) engine, designed to be lightweight, deterministic, and highly scalable. Strongly inspired by the functioning and API of the acclaimed Python library **SimPy**, the project adapts these classic simulation concepts to the modern Go ecosystem. Unlike other libraries that use goroutines and channels for each process, this library uses the new **native Iterators from Go 1.23 (`iter` package)** to manage the execution flow.
Still in pre-alpha version

## 🌟 Technical Highlights

- **Coroutines over Goroutines:** By using `iter.Pull`, each simulated process consumes only the necessary space for its execution stack, without the scheduling overhead of the Go runtime.
- **Non-blocking Simulation Kernel:** The `Environment` manages the virtual time atomically. No network operation or resource blocks the real system thread.
- **Priority Queue with Index Tracking:** Custom implementation of `container/heap` that allows `Fix` (reordering) operations in $O(\log n)$, essential for interruptions and preemption.
- **Deterministic Tie-breaking:** Resources utilize an arrival order counter (`indiceOrd`) to ensure that processes with the same priority are served in strict FIFO order.

---

## 🏗️ System Architecture

The library is divided into four fundamental pillars:

1.  **Environment:** The master clock and event scheduler.
2.  **Process:** The execution unit (Task). It "yields" control to the Environment through commands.
3.  **Commands:** Functions that define the next action of the process (wait time, request resource, signal event).
4.  **PendingAgents:** A flexible interface that allows any object (a Process or a Resource Event) to be placed in waiting queues. This implementation is still under analysis.

---

## 💻 Implementation Examples

The project features complete executable examples in the `exemplos/` folder, demonstrating real usage scenarios:

- **[01_resource_example](./exemplos/01_resource_example/)**: Demonstrates the use of `Environment`, `WaitTime`, and capacity limits through `Resources` (e.g., customers waiting for service at a bank).
- **[02_store_example](./exemplos/02_store_example/)**: Example of the Producer/Consumer pattern using `Store` for secure exchange of packages and data.
- **[03_container_example](./exemplos/03_container_example/)**: Shows the management of continuous/quantitative levels with `Container` (e.g., filling and consuming a reservoir).
- **[04_event_example](./exemplos/04_event_example/)**: Demonstrates synchronization between multiple processes using `Event` (e.g., workers waiting for the arrival of parts).
- **[05_filter_store_example](./exemplos/05_filter_store_example/)**: Demonstrates the use of `FilterStore` with consumers waiting for specific items based on filter functions.
- **[06_network_switch_queues](./exemplos/06_network_switch_queues/)**: Demonstrates advanced usage of `QoSStore` with priority queue balancing (SWRR) in a network scenario (`Switch`).

To run them locally:
```bash
go run ./exemplos/01_resource_example/main.go
go run ./exemplos/02_store_example/main.go
go run ./exemplos/03_container_example/main.go
go run ./exemplos/04_event_example/main.go
go run ./exemplos/05_filter_store_example/main.go
go run ./exemplos/06_network_switch_queues/main.go
```

### Resources and Time
Write your simulation processes sequentially, as if they were real threads, using `yield` to suspend execution:

```go
func myProcess(yield func(Command) bool) {
    // 1. Access the current process (helper to capture context)
    var p *Process
    GetProcess(&p, yield)

    // 2. Wait for virtual time
    yield(WaitTime(10 * time.Second))

    // 3. Try to acquire a limited resource
    // Parameters: process, amount, priority, yield
    res.Acquire(p, 1, 10, yield) 
    
    // The code here only executes when the resource is released for this process
    fmt.Println("Resource acquired at:", p.GetEnv().GetTime())

    yield(WaitTime(5 * time.Second)) // Simulates usage of the resource

    // 4. Release resource
    res.Release(p, 1)
}
```

### Generic Data Queues (`Store[T]`)
Exchange blocking messages between coroutines with strict simulation safety:

```go
func myProducer(yield func(Command) bool) {
    yield(store.Put("Package A", 10)) // Suspends process if Buffer > Configured capacity
}

func myConsumer(yield func(Command) bool) {
    var pkg string
    yield(store.Get(&pkg, 10)) // Suspends process if Buffer == 0
    fmt.Println("Processed:", pkg)
}
```

---

## 🛠️ Feature Status

| Feature | Status | Description |
| :--- | :---: | :--- |
| **Event Loop** | ✅ | Main engine with instantaneous time advancement. |
| **WaitTime** | ✅ | Suspension of processes for a specific duration. |
| **Resources** | ✅ | Capacity management with priority queue and FIFO. |
| **Iter.Pull Integration** | ✅ | Full lifecycle (Created -> Running -> Terminated). |
| **Stores** | ✅ | Generic queue (`Store[T]`) of data/packages with backpressure suspension. |
| **Containers** | ✅ | (Continuous resource management, e.g., fluids, batteries). |
| **Events (Triggers)** | ✅ | (Simple manual success/failure signaling between processes). |
| **FilterStores** | ✅ | Data queues with blocking extraction based on filter functions. |
| **QoSStores** | ✅ | Generic queues optimized for Quality of Service, featuring multiple Traffic Classes via Smooth Weighted Round Robin algorithms. |
| **WaitAny/All** | 🚧 | **Planned** (Complex synchronization of multiple events Condition/AllOf/AnyOf). |
| **Preemptive Resources** | 🚧 | **Planned** (Resources with forced interruption/preemption of low-priority processes). |

---

## 🧩 Components (Extensions)

The project has also started introducing higher-level abstractions for specific simulation domains (e.g. Networks):

- **[Switch](./components/switch.go)** (✅ **Active**): A basic packet router with self-learning MAC tables, backed by `QoSStore` to model output ports and Traffic Class mapping.
- **NetworkLink / Cable** (🚧 **Planned**): Will represent the physical medium connecting network actors (Switches, Hosts), effectively introducing configurable latency and strict bandwidth limits strictly tailored for *Discrete Event Simulation*.

---

## 📦 Requirements

Make sure you are using **Go 1.23** or higher, as the library depends on the iterators functionality (`iter` package) and `range-over-func` support.
