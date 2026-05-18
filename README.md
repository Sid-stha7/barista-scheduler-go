# Barista Scheduler Go

A Go program simulating a coffee shop with priority queuing. It handles high-priority "Live" orders and low-priority "Catering" orders using a biased selection pattern.

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
