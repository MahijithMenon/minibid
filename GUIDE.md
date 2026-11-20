# Build guide (you write the Go; this is the map)

Layout: `cmd/<service>/main.go` (entrypoints, stubs), `internal/openrtb` (types, done), the rest is yours.
Deps you'll `go get`: `github.com/redis/go-redis/v9`, `github.com/segmentio/kafka-go`,
`github.com/aws/aws-sdk-go-v2/{config,service/dynamodb}`, `github.com/prometheus/client_golang`.

## Day 1–2 — auction core (`internal/auction`, in-process fake bidders)
- `type Bidder interface { Name() string; Bid(ctx, *openrtb.BidRequest) (*openrtb.Bid, error) }`
- `Collect(ctx, bidders, req, timeout)`: one goroutine per bidder, **buffered** result channel (size = len(bidders)) so late goroutines never leak, `context.WithTimeout`, select on results vs `ctx.Done()`. Late = timeout, dropped.
- `Decide(bids, floor)` as a **pure function**: drop bids < floor, sort desc, winner pays `max(second, floor)+0.01`, capped at the winner's own bid. Ties: deterministic (by name).
- Table-driven tests: no bids, all below floor, single bid, tie, normal 2nd price, cap. Then `go test -race`.
- Handler `POST /openrtb/bid` (net/http, Go 1.22 mux patterns or chi): decode → validate → Collect → Decide → 200 with BidResponse or 204.

## Day 3 — real bidders
- `cmd/bidder-mock`: env `PORT, MIN_LATENCY_MS, MAX_LATENCY_MS, ERROR_RATE, NO_BID_RATE, MIN_BID, MAX_BID`. Sleep with `select` on `r.Context().Done()`.
- HTTP bidder client implementing `Bidder` (shared `http.Client`, tuned `MaxIdleConnsPerHost`, request built with ctx).
- Run 3 copies on :9001-9003 with different latency/error settings; add Dockerfile services to compose.

## Day 4 — Redis + Kafka
- Freq cap: key `fcap:{user}`; `INCR`, and `EXPIRE` (use `ExpireNX`) when it's the first. Check before auction, record on win. Decide: fail open if Redis is down. Know the limitation: check-then-incr isn't atomic.
- Bidder config in Redis + `GET/PUT /admin/bidders` (token via header).
- Kafka: `kafka-go` Writer, topics `auction.won` / `auction.no_bid`, key = user id, **don't block the request path** (async or channel + worker). Event has a stable `auction_id`.

## Day 5 — event-consumer + DynamoDB Local (:8000)
- Consumer group, `FetchMessage` → process → `CommitMessages` (commit only after success = at-least-once).
- Idempotency: table `wins` PK `user_id`, SK `ts#auction_id`. `TransactWriteItems`: Put win with `attribute_not_exists(user_id)` + update a per-user totals item. `TransactionCanceledException` with `ConditionalCheckFailed` = duplicate → treat as success. Test by replaying an event.
- Create topics/table on startup if missing. Use dummy AWS creds + `BaseEndpoint` for local.

## Day 6 — observability + load test
- Metrics: `bid_requests_total`, `auction_duration_seconds` (histogram, buckets around 100ms), `bidder_outcomes_total{bidder,outcome}`, `auctions_total{result}`. Fill rate = won / requests in PromQL; p99 = `histogram_quantile`.
- Build `deploy/grafana/dashboards/minibid.json` by designing it in the Grafana UI (localhost:3000) and exporting.
- `k6 run loadtest/bid.js`; record real req/s and p99 in README.

## Day 7 — CI, k8s, README
- CI is already in `.github/workflows/ci.yml` (add `.golangci.yml` if you want custom linters).
- `k8s/`: Deployment+Service per service, infra (redis, kafka, dynamodb-local), run on kind.
- README must-haves: diagram, one-command run, load-test numbers, design decisions, limitations, and a line stating AI was used for boilerplate.

Small PRs per day on branches — that's your "code review" talking point.
Ask me to review any day's code or explain a concept when you get stuck.
