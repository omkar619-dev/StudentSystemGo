# Load tests for StudentSystemGo

Two k6 scenarios:

- `scenarios/01-baseline.js` — 1 VU, 60s. Establishes ideal-case latency.
- `scenarios/02-ramp.js` — ramps 5 → 25 → 50 VUs over 5 min. Finds where the system degrades.

## Prerequisites

- k6 installed locally (`winget install k6.k6` on Windows; `brew install k6` on Mac)
- A real exec credential on the target deploy

## Run

```powershell
# Baseline (against EC2 prod)
k6 run -e BASE_URL=http://13.126.165.62 `
       -e LOGIN_EMAIL=your-real-email `
       -e LOGIN_PASSWORD=your-real-password `
       load-tests/scenarios/01-baseline.js

# Ramp
k6 run -e BASE_URL=http://13.126.165.62 `
       -e LOGIN_EMAIL=your-real-email `
       -e LOGIN_PASSWORD=your-real-password `
       load-tests/scenarios/02-ramp.js
```

For local testing, swap to `BASE_URL=http://localhost:3000`.

## What to watch during the test

Open Grafana at `http://localhost:3001` → "StudentSystemGo — Golden Signals" dashboard. Filter by `env=prod`. The 4 panels light up in real time:

- **Traffic**: request rate per path
- **Latency**: p50/p95/p99
- **Errors**: 4xx/5xx rates
- **Saturation**: DB connections, RabbitMQ queue depth

## Interpreting the summary

After the test ends, k6 prints a summary. Key fields:

- `http_req_duration` → overall latency (p50, p95, p99)
- `http_req_failed` → error rate
- `iterations` → total iterations completed
- `vus` → final VU count
- Custom: `endpoint_students_ms`, etc. → per-endpoint trends
