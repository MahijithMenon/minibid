# minibid — real-time bidding service in Go (Kafka, Redis, DynamoDB, Prometheus)

A mini real-time bidding (RTB) auction system modelled on how ad platforms run auctions: an app sends an
OpenRTB bid request, the auction service fans it out to several bidders in parallel, waits a fixed time,
picks a winner with a second-price auction, returns the ad and publishes the result as an event.

> **Status:** work in progress. The scaffolding (infra, CI, OpenRTB types) is in place; the services are being
> built day by day (see [GUIDE.md](GUIDE.md)). Sections marked _TBD_ are filled in as each part lands.
> Load-test numbers are only reported once measured — none are claimed yet.

## Architecture

```
                         ┌──────────────────────────┐
  app / k6 ──POST──────▶ │     auction-service      │──── /metrics ───▶ Prometheus ─▶ Grafana
  /openrtb/bid           │  (Go, net/http)          │
                         └─┬──────┬──────┬───────┬──┘
              fan-out      │      │      │       │ INCR/EXPIRE, bidder config
         (goroutines +     ▼      ▼      ▼       ▼
          ctx timeout)  bidder-1 bidder-2 bidder-3   Redis
                        (mock)   (mock)   (mock)   (freq cap + config)
                                   │
                          auction.won / auction.no_bid
                                   ▼
                                 Kafka
                                   ▼
                          ┌────────────────┐
                          │ event-consumer │──▶ DynamoDB Local (wins, idempotent)
                          └────────────────┘
```

| Service | Role |
|---|---|
| `auction-service` | `POST /openrtb/bid`, `/healthz`, `/metrics`, admin endpoint for bidder config. Parallel bidder calls with a deadline, second-price auction, Redis frequency cap, Kafka publishing. |
| `bidder-mock` | Returns random bids with configurable latency, error rate and no-bid rate, to exercise timeouts and failure handling. Run 2–3 copies with different settings. |
| `event-consumer` | Reads Kafka events and writes win records to DynamoDB, keyed so that replays never double-count. |

## Run it

```bash
docker compose up        # target for the finished project (currently starts infra only)
```

Until the services are containerised, run infra with compose and the Go services natively:

```bash
docker compose up -d redis kafka dynamodb
go run ./cmd/bidder-mock        # ×3 with different env settings
go run ./cmd/auction-service
go run ./cmd/event-consumer
```

Tests and lint: `make test` (runs with `-race`), `make lint`.

Example request (once `auction-service` is implemented):

```bash
curl -s localhost:8080/openrtb/bid -d '{
  "id":"req-1",
  "imp":[{"id":"1","banner":{"w":320,"h":50},"bidfloor":0.5}],
  "app":{"id":"app1","bundle":"com.example.game"},
  "device":{"ifa":"abc","os":"android"},
  "user":{"id":"u1"}
}'
```

## Auction rules

- Each bidder gets the request concurrently and has until the deadline (default 100 ms, lowered by `tmax` if smaller).
- Bidders that are late or fail are dropped, not waited for.
- Bids below the impression's floor price are dropped.
- **Second price:** winner pays `max(second-highest bid, floor) + 0.01`, never more than their own bid.
- Frequency cap: max 3 ads per user per hour (Redis `INCR` + `EXPIRE`).
- Outcomes are published to Kafka as `auction.won` or `auction.no_bid`.
- No bid → HTTP 204.

## Design decisions

- **Second-price auction.** Bidders have no incentive to shade their bid below their true value, and the
  rule is simple to reason about and test. Trade-off: the real market has largely moved to first price.
- **Fixed timeout instead of waiting for all bidders.** An ad slot is only valuable for a few hundred ms and the
  caller has its own deadline, so latency must be bounded by design. Slow bidders cost fill, not latency. The
  result channel is buffered so late goroutines can always finish without leaking.
- **Pure `Decide` function.** Fan-out/IO is separated from the pricing logic, so the auction rules are covered by
  table-driven tests without any goroutines.
- **Redis for frequency caps and config.** `INCR` is atomic and `EXPIRE` gives the sliding hour window cheaply;
  Redis is down → fail open (serve the ad) rather than lose revenue.
- **Async event publishing.** Kafka writes must not sit on the request path; events carry a stable `auction_id`.
- **Idempotent consumer.** Offsets are committed only after a successful write (at-least-once). The write is a
  DynamoDB transaction: put the win with `attribute_not_exists` on key `(user_id, ts#auction_id)` and bump the
  per-user totals. A replayed event hits the condition, the transaction is cancelled, and the consumer treats
  that as success — so totals are never double-counted.

## Observability

Prometheus metrics (planned): `bid_requests_total`, `auction_duration_seconds` (histogram → p50/p95/p99),
`bidder_outcomes_total{bidder,outcome}` (bid / no_bid / timeout / error), `auctions_total{result}`.
Fill rate = won ÷ requests. A Grafana dashboard JSON will live in `deploy/grafana/dashboards/`.

## Load test

_TBD._ Run with `k6 run loadtest/bid.js` and record: hardware, requests/s, p50/p95/p99 latency, error rate.
Everything runs on one laptop, so numbers will be lower and noisier than a real deployment.

| Hardware | Req/s | p50 | p95 | p99 | Errors |
|---|---|---|---|---|---|
| _TBD_ | | | | | |

## Kubernetes / CI

- CI (`.github/workflows/ci.yml`): `go vet`, golangci-lint, `go test -race`, Docker image build per service.
- Kubernetes manifests for kind/minikube: _TBD_ (`k8s/`).
- AWS: DynamoDB and Redis are accessed via AWS SDK v2 / standard clients, so the same code could target real
  DynamoDB and ElastiCache; a deployment to ECS or EKS is a possible next step, not implemented.

## Known limitations

- Simplified OpenRTB 2.5 subset (one impression handled, no seat/deal/currency logic).
- Mock bidders only; no real demand partners.
- Single region, single Kafka broker, single Redis; no HA.
- Frequency cap is check-then-increment, so concurrent requests for one user can briefly overshoot the cap.
- No authentication on the bid endpoint (admin endpoint uses a static token).

## AI assistance

The project scaffolding (compose file, Dockerfile, CI workflow, OpenRTB types, build guide) was generated with
AI help (Claude). The core Go code — auction logic, concurrency, Redis/Kafka/DynamoDB integration — is written
by the repository author.
