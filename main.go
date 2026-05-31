package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Order Types (Service Classes)
const (
	ClassLive      = "live"      // Interactive: high-priority walk-in/app orders
	ClassCatering  = "catering"  // Batch: low-priority bulk office orders
	ClassWholesale = "wholesale" // Heavy Batch: Massive tasks requiring time-slicing
)

// Order represents our incoming request payload
type Order struct {
	ID        int       `json:"id"`
	Item      string    `json:"item"`
	Type      string    `json:"type"` // "live" or "catering"
	CreatedAt time.Time `json:"created_at"`
}

// OrderOrchestrator manages our priority queues and baristas
type OrderOrchestrator struct {
	liveChan     chan Order
	batchChan    chan Order
	workerWg     sync.WaitGroup
	quit         chan struct{}
	orderCounter int
	mu           sync.Mutex
}

func NewOrderOrchestrator(bufferSize int) *OrderOrchestrator {
	return &OrderOrchestrator{
		liveChan:  make(chan Order, bufferSize),
		batchChan: make(chan Order, bufferSize),
		quit:      make(chan struct{}),
	}
}

// StartBaristas spins up our concurrent worker pool
func (oo *OrderOrchestrator) StartBaristas(numWorkers int) {
	fmt.Printf("\033[34m[System]\033[0m Starting %d barista workers...\n", numWorkers)
	for i := 1; i <= numWorkers; i++ {
		oo.workerWg.Add(1)
		go oo.baristaWorker(i)
	}
}

// The core priority engine using a Biased Select Pattern
func (oo *OrderOrchestrator) baristaWorker(id int) {
	defer oo.workerWg.Done()

	for {
		// Absolute Priority Check (Non-blocking)
		// If a live order is waiting, take it immediately and skip everything else.
		select {
		case order := <-oo.liveChan:
			oo.prepare(id, order)
			continue
		default:
		}

		// Balanced/Fallback Check (Blocking)
		select {
		case order := <-oo.liveChan:
			oo.prepare(id, order)

		case order := <-oo.batchChan:
			select {
			case liveOrder := <-oo.liveChan:
				oo.batchChan <- order
				oo.routeOrder(id, liveOrder)
			default:
				oo.routeOrder(id, order)
			}
		case <-oo.quit:
			return
		}
	}
}

func (oo *OrderOrchestrator) routeOrder(baristaID int, o Order) {
	if o.Type == ClassWholesale {
		oo.processHeavyTask(baristaID, o)
	} else {
		oo.prepare(baristaID, o) // prepare handles both Live and Catering
	}
}

// prepare simulates the execution cost of a request
func (oo *OrderOrchestrator) prepare(baristaID int, o Order) {
	queueDuration := time.Since(o.CreatedAt)

	var style string
	if o.Type == ClassLive {
		style = "\033[32m⚡ [LIVE INTERACTIVE]\033[0m"
	} else {
		style = "\033[33m📦 [BATCH CATERING]\033[0m"
	}

	fmt.Printf("\033[36m[Barista %d]\033[0m %s Processing Order #%d (%s) | Wait Time: %v\n",
		baristaID, style, o.ID, o.Item, queueDuration)

	// Simulate work: Catering orders take longer (Head-of-Line Blocking mitigation)
	if o.Type == ClassCatering {
		time.Sleep(100 * time.Millisecond)
	} else {
		time.Sleep(30 * time.Millisecond)
	}
}

// Submit enqueues incoming HTTP payloads into the appropriate channel
func (oo *OrderOrchestrator) Submit(item string, orderType string) Order {
	oo.mu.Lock()
	oo.orderCounter++
	order := Order{
		ID:        oo.orderCounter,
		Item:      item,
		Type:      orderType,
		CreatedAt: time.Now(),
	}
	oo.mu.Unlock()

	if orderType == ClassLive {
		oo.liveChan <- order
	} else {
		oo.batchChan <- order
	}
	return order
}

func (oo *OrderOrchestrator) processHeavyTask(baristaID int, o Order) {
	chunks := 10 // 10 bags of beans to grind

	fmt.Printf("[Barista %d] ⚙️ Starting massive background task: %s\n", baristaID, o.Item)

	for i := 1; i <= chunks; i++ {
		// Do a fraction of the heavy work (The "Slice")
		time.Sleep(1000 * time.Millisecond)
		fmt.Printf("[Barista %d] ⚙️ Finished chunk %d/%d of %s\n", baristaID, i, chunks, o.Item)

		// (Yield) Check if a high-priority order arrived while we were grinding
		select {
		case liveOrder := <-oo.liveChan:
			fmt.Printf("  ⏸️ [INTERRUPT] Barista %d pausing %s to handle a Live Walk-in!\n", baristaID, o.Item)

			// Process the live order immediately
			oo.prepare(baristaID, liveOrder)

			fmt.Printf("  ▶️ [RESUME] Barista %d resuming %s\n", baristaID, o.Item)
		default:
			// No live orders waiting. Safely continue to the next chunk of heavy work.
		}
	}

	fmt.Printf("[Barista %d] Finished massive background task: %s\n", baristaID, o.Item)
}

func main() {
	orchestrator := NewOrderOrchestrator(500)

	// Start with just 1 barista to make the time-slicing highly visible
	orchestrator.StartBaristas(1)

	http.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Item string `json:"item"`
			Type string `json:"type"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if req.Type != ClassLive && req.Type != ClassCatering && req.Type != ClassWholesale {
			http.Error(w, "Invalid type. Must be 'live', 'catering', or 'wholesale'", http.StatusBadRequest)
			return
		}

		order := orchestrator.Submit(req.Item, req.Type)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "queued",
			"order_id": order.ID,
			"type":     order.Type,
		})
	})

	log.Println("[Server] Running on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
