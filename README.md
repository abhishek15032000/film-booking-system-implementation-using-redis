# Cinema Ticket Booking System

Starter code for a practical demonstration of how to construct a robust ticket booking system. This repository highlights the steps needed to solve the classic double-booking problem, handle high contention without race conditions, and provide a good user experience when selecting seats.

## ⚠️ The Problem: Double Bookings

Two users click "Book" on seat A1 at the exact same instant. Only one should win.

```bash
User A ──► read seat A1 → "free" ──► write booking ──► success
User B ──► read seat A1 → "free" ──► write booking ──► ???
```

Without any protection mechanisms, both read the seat as "free", and both writes succeed. Now two people show up for the same seat at the cinema! This is a classic example of a **Race Condition**.

---

## 🏎️ Concurrency & Resolving Race Conditions

A **Race Condition** occurs when multiple processes or threads try to access and modify shared data simultaneously, and the final outcome depends on the unpredictable timing (or "race") of their execution. To fix this, we use concurrency control strategies:

### 1. Pessimistic Concurrency Control (Used right here)
*   **Concept**: *Assume conflicts will happen.* Lock the resource upfront.
*   **How it works**: Before reading or modifying data, the system places a lock on it. No other process can read or write that data until the lock is released.
*   **Examples**: Go's `sync.Mutex`, SQL's `SELECT ... FOR UPDATE`, and our exact Redis `SETNX` implementation.
*   **Pros/Cons**: Guaranteed consistency and no conflicts. However, it can create performance bottlenecks or deadlocks when many users try to access the same resource.

### 2. Optimistic Concurrency Control (OCC)
*   **Concept**: *Assume conflicts are rare.* Don't lock upfront.
*   **How it works**: Records are assigned a version number or timestamp. A process reads the record and its version. When it tries to save an update, it includes this version. The system only accepts the update if the version in the database still matches. If it changed (someone else updated it), the process fails and usually retries.
*   **Examples**: `UPDATE table SET status='booked' WHERE id=1 AND version=2`
*   **Pros/Cons**: Highly performant for read-heavy applications with few conflicts. Under heavy write contention, it results in many rejected requests and retries.

### 3. Queueing / Actor Model (Other methods)
*   **Concept**: Turn concurrent actions into a predictable sequence.
*   **How it works**: All booking requests are funneled into a single Message Queue (e.g., Kafka, RabbitMQ) or an Actor. A single worker processes the queue one by one sequentially.
*   **Pros/Cons**: Eliminates concurrent modifications completely. Harder to implement synchronously (users have to wait asynchronously to find out if they got the ticket).

---

## 🏰 Architecture

Below is a visualization of the internal components and data flow in our Go backend.

```mermaid
graph TD
    Client[Client / Browser]
    API[Go API Server cmd/main.go]
    RedisDB[(Redis Database)]
    
    subgraph Internal Domain
        Service[Booking Service]
        StoreInterface[[Store Interface]]
    end
    
    subgraph Adapters / Implementations
        RedisStore[RedisStore]
        ConcurrentStore[ConcurrentMemoryStore]
        MemStore[MemoryStore]
    end

    Client -- HTTP Requests --> API
    API -- Injects Dep --> Service
    Service -- Uses --> StoreInterface
    
    StoreInterface -. Implements .-> RedisStore
    StoreInterface -. Implements .-> ConcurrentStore
    StoreInterface -. Implements .-> MemStore
    
    RedisStore -- Redis Client --> RedisDB
```

The project follows a modular structure separated into distinct packages:

*   **`cmd/main.go`**: The entry point of the application. It handles the server startup, HTTP request routing, and dependency injection wiring.
*   **`internal/booking/`**: Contains the core domain logic for the booking system.
    *   **Stores**: `MemoryStore`, `ConcurrentMemoryStore`, and `RedisStore` to handle data persistence and concurrency checks.
    *   **Service**: Contains the business logic (`service.go`).
    *   **Handler**: Maps HTTP endpoints to our service logic (`handler.go`).
*   **`internal/adapters/`**: Adapters for external dependencies. Contains the setup for the Redis client connection.
*   **`static/`**: Contains HTML, CSS, or JS files served by the application's base URL route file server.

## 🚀 How to Run the Code

To run this project locally, you need Go and Docker installed.

1.  **Start Redis**:
    Start the Redis server using Docker Compose (assuming you have a `docker-compose.yaml` present):
    ```bash
    docker compose up -d
    ```

2.  **Run the Go API**:
    In a separate terminal, start the Go server:
    ```bash
    go run cmd/main.go
    ```
    The server will start at `http://localhost:8080`.
    *(Note: You can access the UI by going to the base URL in your browser).*

## 🛣️ API Routes

*   `GET /`: Serves the base frontend HTML interface from the `static` directory.
*   `GET /movies`: Returns a list of available movies (e.g., Inception, F1).
*   `GET /movies/{movieID}/seats`: Retrieves a list of booked/held seats for a specific movie so frontend UIs know which seats are blocked.
*   `POST /movies/{movieID}/seats/{seatID}/hold`: Places a temporary lock (a hold) on a given seat for the user. Returns a `sessionID`.
*   `PUT /sessions/{sessionID}/confirm`: Completes the purchase/booking process, changing the temporary hold into a confirmed booking.
*   `DELETE /sessions/{sessionID}`: Explicitly releases a held seat (or it expires eventually based on TTL).

## 🔒 The Solution: How We Prevent Double Bookings

To guarantee that a seat cannot be double-booked across a distributed environment (e.g. if we had multiple Go API servers running), we implement **Pessimistic Locking** backed by Redis.

When two requests attempt to book seat `A1` for movie `Inception` literally microseconds apart, we don't first query if it's free. Instead, the backend goes straight to our `RedisStore`.

Inside `redis_store.go`, we rely on a critically fundamental feature of Redis: **`SETNX` (Set if Not Exists)**.
We build a unique key like: `seat:Inception:A1`.
We command Redis to store the user's booking data into this key **ONLY IF** the key does not already exist.

```go
res := s.rdb.SetArgs(ctx, key, val, redis.SetArgs{
    Mode: "NX", // SET IF NOT EXISTS! THIS IS THE MAGIC.
    TTL:  defaultHoldTTL,
})
```

Because Redis operates on a single execution thread, the very first request to hit Redis gets successfully processed and returns `"OK"`. Every single subsequent request asking to hold that seat will get rejected by Redis immediately.
If the operation returns `OK`, this user won the lock. If it fails, another user holds the seat and our API gracefully rejects the new request with an `ErrSeatAlreadyBooked`, perfectly eliminating data inconsistency issues regardless of how much scale or traffic hits the servers.

## 🧪 Running the Load Tests (Proving the Concept)

We don't just assert that this prevents double-booking; we actively *prove* it with intense automated load testing.

Inside `internal/booking/service_test.go`, there is a test called `TestConcurrentBooking_ExactlyOneWins`.
Anyone can run this test locally using:

```bash
go test -v ./internal/booking -run TestConcurrentBooking_ExactlyOneWins
```

**Test Output Example:**
```bash
=== RUN   TestConcurrentBooking_ExactlyOneWins
2026/04/06 22:59:21 connected to redis at localhost:6379
2026/04/06 22:59:21 Session Booked {4f12cef4-28df-4b13-910c-14d7f17c9a0d screen-1 A1 14f6773d-ba09-49eb-ab10-9e7b819f7381 held ...}
--- PASS: TestConcurrentBooking_ExactlyOneWins (1.44s)
PASS
ok      booking-system/internal/booking 1.503s
```

### What does this test do?

1. It spans **100,000 concurrent Goroutines**, representing an instant spike where 100,000 distinct users are all vigorously attempting to immediately book exact same seat (`screen-1`, `A1`).
2. We utilize a WaitGroup (`sync.WaitGroup`) to ensure all 100k threads crash down on our Redis backend at the exact same split-second.
3. Every goroutine records its outcome into an atomic counter (either a success or a failure).

### The Result We Guarantee

The test asserts structurally that:
1. **Exactly 1** goroutine successfully books the seat.
2. **Exactly 99,999** goroutines fail and receive an error gracefully.

Although having 99,999 "failures" sounds bad on the surface, this is actually the **perfect outcome** for a ticketing system. We preserved guaranteed data integrity under massive immediate contention. No two database entries were created, and nobody was falsely sold tickets that didn't exist.

We have historically compared the `RedisStore` implementation against simpler code like the `MemoryStore`, which relies on standard Go map structures. Under the exact same 100k test without mutex protection, standard slices cause major race conditions, map assignment panics, and massive overlapping duplicate data writes. By migrating to Pessimistic Locking with Redis `SETNX`, or utilizing `sync.RWMutex` (as found in `ConcurrentMemoryStore`), our API achieves perfect multi-user safety.
