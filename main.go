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
	ClassLive     = "live"     // Interactive: high-priority walk-in/app orders
	ClassCatering = "catering" // Batch: low-priority bulk office orders
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
	cateringChan chan Order
	workerWg     sync.WaitGroup
	quit         chan struct{}
	orderCounter int
	mu           sync.Mutex
}

func NewOrderOrchestrator(bufferSize int) *OrderOrchestrator {
	return &OrderOrchestrator{
		liveChan:     make(chan Order, bufferSize),
		cateringChan: make(chan Order, bufferSize),
		quit:         make(chan struct{}),
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

		case order := <-oo.cateringChan:
			// Double-Check Guard: Ensure a live order didn't slip into the channel
			// right as this worker became free.
			select {
			case liveOrder := <-oo.liveChan:
				// Subliminal Bypass: Put the catering order back, save the commuter!
				oo.cateringChan <- order
				oo.prepare(id, liveOrder)
			default:
				oo.prepare(id, order)
			}

		case <-oo.quit:
			return
		}
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
		oo.cateringChan <- order
	}
	return order
}

func main() {
	orchestrator := NewOrderOrchestrator(500)

	// Start with a small worker pool (2 baristas) to make queue constraints visible
	orchestrator.StartBaristas(2)

	// API Endpoint to ingest orders
	http.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Item string `json:"item"`
			Type string `json:"type"` // "live" or "catering"
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if req.Type != ClassLive && req.Type != ClassCatering {
			http.Error(w, "Invalid type. Must be 'live' or 'catering'", http.StatusBadRequest)
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

	log.Println("\033[35m[Server]\033[0m Running on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
