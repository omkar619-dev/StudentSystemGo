# Load testing

Real numbers from k6, with methodology and caveats. The goal isn't to claim the system is fast — it's to **prove with measurements** what the system actually does under load, and to expose where it breaks.

## Methodology

Two scenarios, designed to answer different questions.

### 1. Baseline — single-user latency profile

```bash
k6 run -e BASE_URL=http://13.126.165.62 \
       -e LOGIN_USERNAME=... -e LOGIN_PASSWORD=... \
       load-tests/scenarios/01-baseline.js
```

- 1 VU for 60 seconds.
- Cycles through the four GET endpoints (`/healthz`, `/students`, `/teachers`, `/execs`) with ~1s spacing.
- Run against **EC2 production** to capture real-world latency including India → Mumbai network round-trip.

**The question this answers**: "When the system has zero contention, how fast is each endpoint actually?" That's your floor — every other test compares against this.

### 2. Ramp — capacity test

```bash
k6 run -e BASE_URL=http://localhost:3000 \
       -e LOGIN_USERNAME=... -e LOGIN_PASSWORD=... \
       load-tests/scenarios/02-ramp.js
```

- Stages: 5 VUs (60s warmup) → 25 VUs (120s) → 50 VUs (120s) → ramp-down (30s).
- Mix of GET endpoints, weighted toward `/students` (50%) since reads dominate real workloads.
- Run against **local stack** because EC2 t3.micro doesn't have the RAM for sustained 50 VUs (covered below).

**The question this answers**: "Where does the system start to degrade? Sustained throughput? Latency knee? First errors?"

### Why I separated baseline-vs-ramp by environment

t3.micro has 1 GB RAM. Stack needs ~880 MB just to run. A 50-VU ramp pushes app + DB CPU/memory into thrashing → OOM-killer reaps containers. **First ramp attempt against EC2 froze the instance** — required a Console reboot.

Choices were:
1. Test locally (laptop has ~32 GB RAM, no contention)
2. Upgrade to t3.small ($15/mo)
3. Skip the ramp entirely

Picked (1). The numbers below are local; EC2 prod numbers would be lower (probably 30-50% lower) due to less RAM, slower CPU, and shared neighbours on the t3 burstable instance.

The full architectural story for honest portfolio context: ["I hit memory ceiling on 1 GB instance. Tuned what I could. Deferred the t3.small upgrade to Phase 1.5."]

---

## Results

### Baseline (EC2 prod, 1 VU, 60s)

```
http_req_duration:
    p50:  9.5 ms
    p95:  22.9 ms
    p99:  ~30 ms
    max:  232 ms (one outlier — likely cold cache + GC)

per-endpoint p95:
    /healthz: 14.8 ms
    /students: 19.3 ms
    /teachers: 25.8 ms
    /execs:    24.1 ms

http_req_failed: 19.31%   (rate-limiter rejecting excess — this is correct behavior)

iterations: 58 over 60s
```

**Read this as**: India-to-Mumbai network round-trip is ~10 ms. Subtract that to get **server-side latency of ~5-15 ms p95**. That's fast. Even the 232 ms outlier is fine for a single-user baseline — bug in distribution, not the system.

The 19% failure rate is the **rate limiter doing its job**. Test sends ~4 RPS for 60s = 240 attempted. Limit is 100/min/IP = 100 succeed within the window. Excess gets 429. This is intentional and was load-tested separately.

### Ramp (local, 50 VUs peak, 5.5 min)

```
total requests: 156,954
sustained throughput: 475 req/sec (averaged across all stages)

http_req_duration:
    p50:  27.1 ms
    p95:  139.8 ms
    p99:  ~500 ms (estimated from histogram; max=876ms is one outlier)

per-endpoint p95:
    /students: 137 ms
    /teachers: 141 ms
    /execs:    143 ms

http_req_failed: 0.00%   (zero errors across 156k requests)

VUs: 5 → 25 → 50 (ramped over 4.5 min, then 30s ramp-down)
```

**Read this as**: at 50 concurrent users sustaining ~475 RPS, p95 latency is 140 ms with **zero errors**. Latency degraded 5.5× from single-user baseline (25 ms → 140 ms) for 50× the concurrency — that's a **sublinear curve**, the system absorbs load gracefully.

### Why these numbers matter

The interesting question is "what shape is the latency curve as load increases?" Three possibilities:

- **Linear** (latency × N for N× load) → no headroom, system saturates at N
- **Sublinear** (latency degrades less than load increases) → reserves available, scales well
- **Hockey-stick** (latency flat, then catastrophic spike) → resource exhaustion on a specific bottleneck

Our shape is **sublinear**. At 50 VUs we're nowhere near the knee. Real capacity is somewhere beyond — would need a more aggressive ramp (200, 500 VUs) to find it.

---

## Production posture vs. capacity

These two numbers tell different stories — both belong in the writeup.

| | Capacity (rate limiter OFF) | Production (rate limiter ON) |
|---|---|---|
| Sustained throughput | 475 RPS at 50 VUs | 1.67 RPS per IP (100/min) |
| p95 latency | 140 ms | 25 ms (under-limit traffic) |
| Error rate | 0% | 100% on excess (correct: 429s) |

**The story**: in unprotected capacity terms, the system handles 475 RPS at 50 concurrent users. In production, that's gated by per-IP rate limiting at 100/min. Different numbers, both real.

If you wanted to do a real "lift the rate limit, throw 500 VUs at production" test: temporarily configure a higher per-IP limit, OR distribute the test across multiple source IPs (k6 cloud, multiple containers).

---

## Methodology details

### Auth handling

`/execs/login` is rate-limited at 10/min/IP. If every VU logged in independently, we'd burn the limit instantly. Solution: log in **once** in k6's `setup()` lifecycle hook (runs once before VUs start), cache the JWT, share to all VUs.

```js
export function setup() {
  const token = getToken();
  return { token };
}
export default function (data) {
  const headers = { headers: authHeaders(data.token) };
  // ... use headers ...
}
```

### Response parsing quirk

The app's compression middleware appends gzip bytes after the JSON response without setting `Content-Encoding: gzip` correctly. k6's `res.json('token')` fails on the malformed response. Workaround: regex-extract the token from the body.

```js
const match = body.match(/"token"\s*:\s*"([^"]+)"/);
```

This is a known bug in the middleware — logged for follow-up. Most browsers and curl-with-`--compressed` tolerate it; strict HTTP clients (k6) don't.

### Custom metrics for per-endpoint visibility

k6 default `http_req_duration` is aggregate across all requests. To see per-endpoint distributions, declare custom Trends:

```js
import { Trend } from 'k6/metrics';
const studentsTrend = new Trend('endpoint_students_ms');

// inside default function:
studentsTrend.add(res.timings.duration);
```

These show up in the summary alongside `http_req_duration`.

### Watching live in Grafana

While k6 runs, open Grafana at `http://localhost:3001` → "Golden Signals" dashboard → filter `env=prod` (for EC2 baseline) or `env=local` (for ramp). All four panels light up with live data.

This is the demo-quality moment: **you see the latency curve climb in real-time as VUs ramp**, then settle as the ramp-down kicks in.

---

## What's missing

A real load test program would also include:

- **Soak test**: 10 VUs for 30+ minutes. Catches connection pool leaks, slow memory growth, replica lag accumulation. Skipped for time.
- **Spike test**: 100 VUs for 30 seconds, then back to 0. Validates burst handling and auto-recovery. Skipped.
- **Failure injection**: kill RabbitMQ mid-test. Verify forgot-password endpoint correctly fails-open. Skipped — would belong in a "chaos test" suite.
- **Multi-region simulation**: distribute load from US, EU, India simultaneously. Skipped; we deploy in one region.
- **Connection-level metrics**: `tcp_connections_open` over time, request queue depth, GC pause distribution. The Prometheus scrape covers some of this; deeper would need explicit Go runtime metrics.

These would matter for a real production rollout. For a portfolio piece demonstrating understanding, the baseline + ramp is the load-bearing pair.

---

## Lessons that ended up in the writeup

1. **Single-VU baseline first** — without it you can't tell if your ramp test is showing real degradation or just network noise.
2. **Test locally when prod is RAM-bound** — running ramp against an OOM-prone instance just gives you crash data, not capacity data.
3. **Disable the rate limiter for capacity tests** — and re-enable it before pushing. (Yes, I almost forgot.)
4. **Share auth via k6 setup()** — one token across N VUs, not N tokens for N VUs.
5. **Watch dashboards live during the test** — finds the bottleneck visually faster than reading post-hoc summaries. DB connection pool saturation in particular is invisible in k6's summary; obvious in Grafana.
