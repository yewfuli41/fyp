// Concurrency / race-condition test: fires N customers at the exact same
// bookable slot at (as close to) the same instant, to prove the system can
// never let two of them win the same slot — no matter how the mocked Go unit
// tests already model it. This complements (does not replace) the unit test
// for "accepting a booking rejects other pending requests for the same slot"
// (see internal/service/booking_service_test.go), which only proves the
// *sequential* accept-time logic works — it can't prove anything about two
// requests racing to *create* a booking on the same slot at once. That race
// is instead guarded by the `uq_active_booking_per_slot` unique index in
// database/schema.sql, and this script is what actually exercises it under
// real concurrent load.
//
// Usage:
//   BASE_URL=http://localhost:8080/query VUS=10 k6 run double_booking_race.js
//
// Prerequisites:
//   - The Go server is running and reachable at BASE_URL (default
//     http://localhost:8080/query).
//   - Its database is a disposable/test database — this script creates a
//     real business, service, slot, and VUS customer accounts on every run.
//     Do NOT point BASE_URL at a shared/production environment.

import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/query';
const VU_COUNT = Number(__ENV.VUS || 10);

const successfulBookings = new Counter('successful_bookings');
const cleanRejections = new Counter('already_booked_rejections');
const unexpectedErrors = new Counter('unexpected_errors');

export const options = {
  scenarios: {
    race: {
      executor: 'per-vu-iterations',
      vus: VU_COUNT,
      iterations: 1,
      maxDuration: '30s',
    },
  },
  thresholds: {
    // The whole point of the test: exactly one of the VU_COUNT concurrent
    // attempts may ever succeed in booking the single available slot.
    successful_bookings: ['count==1'],
    unexpected_errors: ['count==0'],
  },
};

function gql(query, variables, token) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const res = http.post(BASE_URL, JSON.stringify({ query, variables }), { headers });
  return { res, body: safeJson(res) };
}

function safeJson(res) {
  try {
    return res.json();
  } catch (e) {
    return null;
  }
}

function requireToken(result, label) {
  const token = result.body && result.body.data && result.body.data.signUp && result.body.data.signUp.token;
  if (!token) {
    throw new Error(`${label} failed: ${result.res.status} ${result.res.body}`);
  }
  return token;
}

// setup() runs once, single-threaded, before any VU starts iterating. It
// creates one business owner + business + service (one option) + one
// owner-managed slot offering only that option — the single seat every VU
// below will race for — plus one distinct, freshly-signed-up customer
// account per VU, so each VU races as its own real authenticated user
// rather than sharing a token.
export function setup() {
  const stamp = Date.now();

  const owner = gql(
    `mutation($u: SignUpInput!) { signUp(user: $u) { token } }`,
    { u: { username: `K6Owner${stamp}`, email: `owner-${stamp}@k6test.local`, contactNumber: '0123456789', password: 'Password123' } }
  );
  const ownerToken = requireToken(owner, 'owner signUp');

  const allDayHours = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday']
    .map((day) => ({ day, startTime: '2000-01-01T00:00:00Z', endTime: '2000-01-01T23:59:00Z' }));

  const business = gql(
    `mutation($b: BusinessProfileInput!) { registerBusinessProfile(business: $b) { businessId } }`,
    {
      b: {
        businessName: `K6 Race Biz ${stamp}`,
        address: '1 Test Street',
        businessContactNumber: '0123456789',
        businessEmail: `biz-${stamp}@k6test.local`,
        workingHours: allDayHours,
      },
    },
    ownerToken
  );
  if (!business.body || !business.body.data || !business.body.data.registerBusinessProfile) {
    throw new Error(`registerBusinessProfile failed: ${business.res.status} ${business.res.body}`);
  }

  const service = gql(
    `mutation($s: ServiceInput!) {
      createService(service: $s) { serviceId serviceOptions { serviceOptionId } }
    }`,
    {
      s: {
        serviceName: `K6 Race Service ${stamp}`,
        serviceOptions: [
          {
            serviceOptionName: 'Standard',
            serviceOptionItems: [{ serviceOptionItemName: 'Standard item' }],
            effectiveFrom: new Date().toISOString().slice(0, 10),
          },
        ],
      },
    },
    ownerToken
  );
  const serviceOptionId =
    service.body &&
    service.body.data &&
    service.body.data.createService &&
    service.body.data.createService.serviceOptions[0] &&
    service.body.data.createService.serviceOptions[0].serviceOptionId;
  if (!serviceOptionId) {
    throw new Error(`createService failed: ${service.res.status} ${service.res.body}`);
  }

  const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
  const slot = gql(
    `mutation($i: ServiceSlotInput!) {
      createServiceSlot(input: $i) { serviceSlotId serviceSlotOptions { slotOptionId } }
    }`,
    {
      i: {
        date: tomorrow,
        startTime: '2000-01-01T14:00:00Z',
        endTime: '2000-01-01T15:00:00Z',
        serviceOptionIds: [serviceOptionId],
      },
    },
    ownerToken
  );
  const slotOptionId =
    slot.body &&
    slot.body.data &&
    slot.body.data.createServiceSlot &&
    slot.body.data.createServiceSlot.serviceSlotOptions[0] &&
    slot.body.data.createServiceSlot.serviceSlotOptions[0].slotOptionId;
  if (!slotOptionId) {
    throw new Error(`createServiceSlot failed: ${slot.res.status} ${slot.res.body}`);
  }

  const customerTokens = [];
  for (let i = 0; i < VU_COUNT; i++) {
    const customer = gql(
      `mutation($u: SignUpInput!) { signUp(user: $u) { token } }`,
      { u: { username: `K6Cust${stamp}${i}`, email: `customer-${stamp}-${i}@k6test.local`, contactNumber: '0123456789', password: 'Password123' } }
    );
    customerTokens.push(requireToken(customer, `customer ${i} signUp`));
  }

  return { slotOptionId, customerTokens };
}

// default(): one call per VU (per-vu-iterations, 1 iteration each), so all
// VU_COUNT customers fire createBooking against the same slotOptionId as
// close to simultaneously as k6 can start them — the actual race window this
// test is targeting.
export default function (data) {
  const token = data.customerTokens[__VU - 1];
  const { res, body } = gql(
    `mutation($slotOptionId: ID!) {
      createBooking(slotOptionId: $slotOptionId) { bookingId status }
    }`,
    { slotOptionId: data.slotOptionId },
    token
  );

  const succeeded = !!(body && body.data && body.data.createBooking && body.data.createBooking.bookingId);
  // Validation errors from this API are wrapped: the top-level error message
  // is always the generic "validation failed", with the real per-field
  // message(s) nested in extensions.validationErrors[].message (see
  // graph/graphErrs/validation_error.go) — so both places need checking.
  const EXPECTED_REJECTION = /no longer available|already been booked/i;
  const rejectedAsExpected = !!(
    body &&
    body.errors &&
    body.errors.some((e) => {
      if (EXPECTED_REJECTION.test(e.message)) return true;
      const validationErrors = e.extensions && e.extensions.validationErrors;
      return Array.isArray(validationErrors) && validationErrors.some((ve) => EXPECTED_REJECTION.test(ve.message));
    })
  );

  check(res, {
    'either booked successfully or cleanly rejected as unavailable': () => succeeded || rejectedAsExpected,
  });

  if (succeeded) {
    successfulBookings.add(1);
  } else if (rejectedAsExpected) {
    cleanRejections.add(1);
  } else {
    unexpectedErrors.add(1);
    console.error(`VU ${__VU} got an unexpected response (status ${res.status}): ${res.body}`);
  }
}

export function handleSummary(data) {
  const count = (name) => (data.metrics[name] ? data.metrics[name].values.count : 0);
  return {
    stdout: `
Double-booking race test summary
---------------------------------
Concurrent customers (VUs): ${VU_COUNT}
Successful bookings:        ${count('successful_bookings')}  (expected: exactly 1)
Rejected as already booked: ${count('already_booked_rejections')}  (expected: ${VU_COUNT - 1})
Unexpected errors:          ${count('unexpected_errors')}  (expected: 0)
${count('successful_bookings') === 1 && count('unexpected_errors') === 0 ? 'RESULT: PASS — the slot was never double-booked.' : 'RESULT: FAIL — see above.'}
`,
  };
}
