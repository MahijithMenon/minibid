// k6 run loadtest/bid.js  -- fill in once the auction-service works (Day 6)
import http from 'k6/http';
import { check } from 'k6';
export const options = { vus: 50, duration: '30s' };
export default function () {
  const body = JSON.stringify({
    id: `${__VU}-${__ITER}`,
    imp: [{ id: '1', banner: { w: 320, h: 50 }, bidfloor: 0.5 }],
    app: { id: 'app1' }, device: { ifa: 'x' }, user: { id: `u${Math.floor(Math.random()*10000)}` },
  });
  const r = http.post('http://localhost:8080/openrtb/bid', body, { headers: { 'Content-Type': 'application/json' } });
  check(r, { ok: (r) => r.status === 200 || r.status === 204 });
}
