import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 200,
  duration: '10s',
};

export default function () {
  const eventId = __ENV.EVENT_ID || '00000000-0000-0000-0000-000000000000';
  const userId = `user-${__ITER}`;
  const idem = `idem-${__VU}-${__ITER}`;
  const res = http.post(`http://localhost:8080/v1/events/${eventId}/book`, JSON.stringify({ user_id: userId, seats: [], idempotency_key: idem }), { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'status ok': r => [200,202,409].includes(r.status) });
  sleep(0.1);
}


