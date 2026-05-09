// Ramp test — slowly increase load and watch where things break.
// Goal: find the request rate where p95 latency degrades or errors appear.
//
// Stages: 5 → 25 → 50 VUs over 5 minutes.
// Skips POST /execs/login (rate-limited at 10/min — would be the bottleneck
// instead of the actual app).
//
// Run:
//   k6 run -e BASE_URL=http://13.126.165.62 \
//          -e LOGIN_EMAIL=... -e LOGIN_PASSWORD=... \
//          load-tests/scenarios/02-ramp.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { getToken, authHeaders, BASE_URL } from '../helpers/auth.js';

export const options = {
  stages: [
    { duration: '60s',  target: 5  },   // warmup at 5 VUs
    { duration: '120s', target: 25 },   // ramp to 25
    { duration: '120s', target: 50 },   // ramp to 50
    { duration: '30s',  target: 0  },   // ramp down (cooldown)
  ],
  thresholds: {
    // Looser bars during ramp — system is supposed to slow down here.
    'http_req_duration': ['p(95)<2000'],   // p95 under 2s
    'http_req_failed':   ['rate<0.05'],    // <5% errors
  },
};

const errorRate = new Rate('errors');
const studentsTrend = new Trend('endpoint_students_ms');
const teachersTrend = new Trend('endpoint_teachers_ms');
const execsTrend    = new Trend('endpoint_execs_ms');

export function setup() {
  return { token: getToken() };
}

export default function (data) {
  const headers = { headers: authHeaders(data.token) };

  // Mix of endpoints — weighted toward reads (the realistic case).
  // ~50% /students, ~25% /teachers, ~25% /execs.
  const r = Math.random();
  let res;

  if (r < 0.5) {
    res = http.get(`${BASE_URL}/students?limit=10`, headers);
    studentsTrend.add(res.timings.duration);
  } else if (r < 0.75) {
    res = http.get(`${BASE_URL}/teachers?limit=10`, headers);
    teachersTrend.add(res.timings.duration);
  } else {
    res = http.get(`${BASE_URL}/execs?limit=10`, headers);
    execsTrend.add(res.timings.duration);
  }

  const ok = check(res, {
    'status 200': (r) => r.status === 200,
  });
  errorRate.add(!ok);

  // Don't sleep — let VUs hammer as fast as possible.
  // VUs are already gated by network round-trip + server processing.
}
