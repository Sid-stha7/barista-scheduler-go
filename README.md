##  Project Overview: Barista Order Scheduler

**Barista Order Scheduler** is a high-performance, concurrent HTTP microservice built in Go that demonstrates component-level latency mitigation strategies outlined in Google's foundational paper, **"The Tail at Scale"**. 


* **Title:** The Tail at Scale
* **Authors:** Jeffrey Dean and Luiz André Barroso (Google)
* **Publication:** Communications of the ACM (2013)
* **Core Concept Implemented:** *Differentiating Service Classes and Higher-Level Queuing Management (Section 4, Paragraph 2).*

 [Read the Full Google Research Paper Here](https://research.google/pubs/the-tail-at-scale/)

The project models a real world architectural challenge preventing heavy background batch processes from degrading high priority interactive user experiences using an intuitive **Coffee Shop Analogy** (Live Wal In Commuters vs. Bulk Office Catering Orders).

When a massive flood of asynchronous background work hits the system, a traditional First-In, First-Out (FIFO) queue causes severe **Head-of-Line (HoL) Blocking**, driving up p99/p99.9 tail latency for active users. This orchestrator eliminates that micro variability by implementing **Differentiated Service Classes** natively at the application layer.

###  Key Architectural Highlights

* **Dual-Channel Isolation:** Completely separates incoming workloads into parallel, thread safe memory buffers (`liveChan` and `cateringChan`) to eliminate memory contention.
* **Biased Select Pattern:** Leverages Go's native concurrency primitives to intentionally break default channel fairness, ensuring worker goroutines (baristas) prioritize interactive traffic first.
* **Double-Check Concurrency Guard:** Implements a nested, non-blocking check right before executing lower priority tasks. This prevents context-switching delays if an interactive user request arrives at the exact microsecond a worker becomes free.
* **Zero-Allocation Backpressure:** Minimizes heap allocation overhead during peak traffic spikes by utilizing pre-buffered ring channels, keeping execution times highly deterministic.


## How to Run

1.  Make sure you have Go installed.
2.  Run the server:
    ```bash
    go run main.go
    ```
    The server will start on `http://localhost:8080`.

## How to Test

You can use `curl` to send orders to the server. Open a new terminal window to run these commands.

### 1. Send a Live Order (High Priority)
```bash
curl -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -d '{"item": "Latte", "type": "live"}'
```

### 2. Send a Catering Order (Low Priority)
```bash
curl -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -d '{"item": "10x Cappuccinos", "type": "catering"}'
```

### 3. Test Priority Logic
To see the priority logic in action, you can send many catering orders and then a live order.

On Mac/Linux, you can run this command to flood the server with 20 catering orders: (!Note:This will create a queue of 20 catering orders, which will be processed after the live orders in the queue, You can change the number of catering orders by changing the number in the for loop, eg. change 20 to 100 to flood the server with 100 catering orders)
```bash
for i in {1..20}; do
  curl -X POST http://localhost:8080/order -H "Content-Type: application/json" -d "{\"item\": \"Bulk Latte $i\", \"type\": \"catering\"}" &
done
```

While that is running or right after, send a live order:
```bash
curl -X POST http://localhost:8080/order -H "Content-Type: application/json" -d '{"item": "VIP Espresso", "type": "live"}'
```

You should see in the server logs that the `⚡ [LIVE INTERACTIVE]` order gets prioritized or processed quickly despite the queue of `📦 [BATCH CATERING]` orders.

## Output Colors
- **[System]**: Blue
- **[Server]**: Magenta
- **[Barista]**: Cyan
- **Live Orders**: Green
- **Catering Orders**: Yellow



