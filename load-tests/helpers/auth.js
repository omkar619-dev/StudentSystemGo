// Auth helper for k6 — login once, share the token across all VUs.
//
// Why this exists: /execs/login is rate-limited at 10/min. If every VU
// logged in independently, we'd burn through the limit instantly. Instead,
// we log in ONCE in the setup phase and pass the token to every VU.
//
// k6 has a `setup()` lifecycle hook that runs once before VUs start.
// Whatever it returns is passed as the first arg to default function.

import http from 'k6/http';
import { check, fail } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://13.126.165.62';
const LOGIN_USERNAME = __ENV.LOGIN_USERNAME || '';
const LOGIN_PASSWORD = __ENV.LOGIN_PASSWORD || '';

export function getToken() {
  if (!LOGIN_USERNAME || !LOGIN_PASSWORD) {
    fail('Set LOGIN_USERNAME and LOGIN_PASSWORD env vars (use a real exec credential).');
  }

  const res = http.post(
    `${BASE_URL}/execs/login`,
    JSON.stringify({ username: LOGIN_USERNAME, password: LOGIN_PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(res, { 'login status 200': (r) => r.status === 200 }) || fail(`login failed: ${res.status} ${res.body}`);

  // The server's compression middleware appends gzip bytes after the JSON
  // (no Content-Encoding header, hard to decode cleanly). So we regex-extract
  // the token from the partial JSON we can see at the start of the body.
  const body = res.body.toString();
  const match = body.match(/"token"\s*:\s*"([^"]+)"/);
  if (!match) fail(`login response missing token field. Body starts with: ${body.substring(0, 100)}`);
  return match[1];
}

export function authHeaders(token) {
  return {
    'Cookie': `Bearer=${token}`,
    'Accept-Encoding': 'gzip',
  };
}

export { BASE_URL };
