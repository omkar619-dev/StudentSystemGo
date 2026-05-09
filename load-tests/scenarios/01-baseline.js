// Baseline test — 1 VU, 60 seconds.
// Goal: establish p50/p95/p99 latency for each endpoint at ZERO contention.
// This is the "best case" reference all other tests compare against.
//
// Run:
//   k6 run -e BASE_URL=http://13.126.165.62 \
//          -e LOGIN_EMAIL=... -e LOGIN_PASSWORD=... \
//          load-tests/scenarios/01-baseline.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { getToken, authHeaders, BASE_URL } from '../helpers/auth.js';

export const options = {
  vus: 1,
  duration: '60s',
  // Custom thresholds — k6 fails the test if any are violated.
  // Tune these once you know real numbers; first run is exploratory.
  thresholds: {
    'http_req_duration': ['p(95)<500'],          // p95 under 500ms across all requests
    'http_req_failed':   ['rate<0.01'],           // <1% errors
  },
};

// Per-endpoint latency trends (so we can see them split out in the summary).
const healthzTrend  = new Trend('endpoint_healthz_ms');
const studentsTrend = new Trend('endpoint_students_ms');
const teachersTrend = new Trend('endpoint_teachers_ms');
const execsTrend    = new Trend('endpoint_execs_ms');

// k6 lifecycle: setup runs ONCE before VUs start.
// Returns a value passed to the default function as `data`.
export function setup() {
  const token = getToken();
  console.log('setup: got token, kicking off VUs');
  return { token };
}

export default function (data) {
  const headers = { headers: authHeaders(data.token) };

  // /healthz — cheapest endpoint, no DB
  let res = http.get(`${BASE_URL}/healthz`, headers);
  check(res, { 'healthz 200': (r) => r.status === 200 });
  healthzTrend.add(res.timings.duration);

  // GET /students — DB read (replica)
  res = http.get(`${BASE_URL}/students?limit=10`, headers);
  check(res, { 'students 200': (r) => r.status === 200 });
  studentsTrend.add(res.timings.duration);

  // GET /teachers — DB read (replica)
  res = http.get(`${BASE_URL}/teachers?limit=10`, headers);
  check(res, { 'teachers 200': (r) => r.status === 200 });
  teachersTrend.add(res.timings.duration);

  // GET /execs — DB read + cache (Redis cache-aside)
  res = http.get(`${BASE_URL}/execs?limit=10`, headers);
  check(res, { 'execs 200': (r) => r.status === 200 });
  execsTrend.add(res.timings.duration);

  // Pace ourselves — ~1 second between iterations.
  // At 1 VU + 4 endpoints + ~1s sleep, we hit ~4 RPS.
  sleep(1);
}
