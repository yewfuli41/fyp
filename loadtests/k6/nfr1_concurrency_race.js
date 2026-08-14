// NFR-1 concurrency test:
//   "The system shall handle at least 50 concurrent booking and rescheduling
//   requests while maintaining data consistency, ensuring that no appointment
//   slot is assigned to more than one active booking."
//
// double_booking_race.js proves the guarantee for the simplest possible case
// (one slot, one operation type, one race). This script exercises the fuller
// requirement: multiple slots contended at once (default 5), a MIX of two
// different operations racing for the same slot (createBooking for brand-new
// customers, rescheduleBooking for customers moving an existing booking),
// and a total load of 50+ concurrent requests (default 5 slots x 10
// attempts/slot = 50, configurable via SLOTS / ATTEMPTS_PER_SLOT).
//
// Usage:
//   k6 run nfr1_concurrency_race.js
//   SLOTS=8 ATTEMPTS_PER_SLOT=10 k6 run nfr1_concurrency_race.js   # 80 concurrent requests
//
// Prerequisites: same as double_booking_race.js — a running server pointed
// at a disposable/test database. Never point BASE_URL at shared/production
// data; this script creates real accounts, a business, and many slots.

import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/query';
const SLOTS = Number(__ENV.SLOTS || 5);
const ATTEMPTS_PER_SLOT = Number(__ENV.ATTEMPTS_PER_SLOT || 10);
const TOTAL_VUS = SLOTS * ATTEMPTS_PER_SLOT;

const successfulLandings = new Counter('successful_landings');
const cleanRejections = new Counter('already_booked_rejections');
const unexpectedErrors = new Counter('unexpected_errors');

export const options = {
  scenarios: {
    race: {
      executor: 'per-vu-iterations',
      vus: TOTAL_VUS,
      iterations: 1,
      maxDuration: '60s',
    },
  },
  thresholds: {
    // The core invariant: exactly one winner per contended slot, never more,
    // never fewer (a genuine validation/setup bug would show up as fewer).
    successful_landings: [`count==${SLOTS}`],
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

function signUpCustomer(stamp, label) {
  const result = gql(
    `mutation($u: SignUpInput!) { signUp(user: $u) { token } }`,
    { u: { username: `K6${label}${stamp}`, email: `${label}-${stamp}@k6test.local`, contactNumber: '0123456789', password: 'Password123' } }
  );
  return requireToken(result, `${label} signUp`);
}

// setup() runs once, single-threaded, before any VU starts iterating. It
// builds:
//   - one business owner + business + one service option
//   - SLOTS "target" slots (all on the same future date, one hour apart) —
//     the contended slots every attempt below races for
//   - for each attempt that will be a reschedule (roughly half of each
//     slot's ATTEMPTS_PER_SLOT), a distinct "origin" slot (its own future
//     date, so it can never collide with a target slot) plus a customer who
//     already holds a booking there — the booking this attempt will try to
//     move onto a target slot
//   - for each attempt that will be a create (the other half), a fresh
//     customer with no existing booking
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
        businessName: `K6 NFR1 Biz ${stamp}`,
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
        serviceName: `K6 NFR1 Service ${stamp}`,
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

  function createSlot(dateOffsetDays, hour) {
    const date = new Date(Date.now() + dateOffsetDays * 24 * 60 * 60 * 1000).toISOString().slice(0, 10);
    const hh = String(hour).padStart(2, '0');
    const slot = gql(
      `mutation($i: ServiceSlotInput!) {
        createServiceSlot(input: $i) { serviceSlotOptions { slotOptionId } }
      }`,
      {
        i: {
          date,
          startTime: `2000-01-01T${hh}:00:00Z`,
          endTime: `2000-01-01T${hh}:59:00Z`,
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
    return slotOptionId;
  }

  // Target slots: all tomorrow, one hour apart starting at 08:00 — the
  // contended slots every attempt below races for.
  const targetSlots = [];
  for (let i = 0; i < SLOTS; i++) {
    targetSlots.push(createSlot(1, 8 + i));
  }

  // Build one attempt spec per VU, round-robin across target slots so each
  // slot gets exactly ATTEMPTS_PER_SLOT contenders, alternating create vs
  // reschedule within each slot's group.
  const attempts = [];
  let originDayOffset = 2; // origin slots start the day after target slots, one distinct date each
  for (let i = 0; i < TOTAL_VUS; i++) {
    const targetSlotOptionId = targetSlots[i % SLOTS];
    const localIndex = Math.floor(i / SLOTS);
    const isReschedule = localIndex % 2 === 1;

    if (!isReschedule) {
      const token = signUpCustomer(stamp, `create${i}`);
      attempts.push({ type: 'create', token, targetSlotOptionId });
      continue;
    }

    const token = signUpCustomer(stamp, `resched${i}`);
    const originSlotOptionId = createSlot(originDayOffset, 10);
    originDayOffset += 1;
    const booking = gql(
      `mutation($slotOptionId: ID!) { createBooking(slotOptionId: $slotOptionId) { bookingId } }`,
      { slotOptionId: originSlotOptionId },
      token
    );
    const bookingId = booking.body && booking.body.data && booking.body.data.createBooking && booking.body.data.createBooking.bookingId;
    if (!bookingId) {
      throw new Error(`origin createBooking failed for attempt ${i}: ${booking.res.status} ${booking.res.body}`);
    }
    attempts.push({ type: 'reschedule', token, bookingId, targetSlotOptionId });
  }

  return { ownerToken, targetSlots, attempts };
}

// default(): one call per VU (per-vu-iterations, 1 iteration each), so all
// TOTAL_VUS attempts fire concurrently — a mix of createBooking and
// rescheduleBooking, all racing across SLOTS contended target slots at once.
export default function (data) {
  const attempt = data.attempts[__VU - 1];

  const { res, body } =
    attempt.type === 'create'
      ? gql(
          `mutation($slotOptionId: ID!) { createBooking(slotOptionId: $slotOptionId) { bookingId status } }`,
          { slotOptionId: attempt.targetSlotOptionId },
          attempt.token
        )
      : gql(
          `mutation($bookingId: ID!, $newSlotOptionId: ID!) {
            rescheduleBooking(bookingId: $bookingId, newSlotOptionId: $newSlotOptionId) { bookingId status }
          }`,
          { bookingId: attempt.bookingId, newSlotOptionId: attempt.targetSlotOptionId },
          attempt.token
        );

  const succeeded = !!(
    body &&
    body.data &&
    ((body.data.createBooking && body.data.createBooking.bookingId) ||
      (body.data.rescheduleBooking && body.data.rescheduleBooking.bookingId))
  );

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
    'either landed successfully or was cleanly rejected as unavailable': () => succeeded || rejectedAsExpected,
  });

  if (succeeded) {
    successfulLandings.add(1);
  } else if (rejectedAsExpected) {
    cleanRejections.add(1);
  } else {
    unexpectedErrors.add(1);
    console.error(`VU ${__VU} (${attempt.type} -> slot ${attempt.targetSlotOptionId}) got an unexpected response (status ${res.status}): ${res.body}`);
  }
}

// teardown() runs once, single-threaded, after all VUs finish. It is the
// independent, server-side check that backs up the client-side tally above:
// query the business's own booking list and confirm, for every target slot,
// that exactly one non-cancelled/non-rejected booking actually landed there
// — this is what the "no slot assigned to more than one active booking"
// half of NFR-1 is actually asserting, verified against real stored state
// rather than just counting HTTP responses.
export function teardown(data) {
  const { res, body } = gql(
    `query {
      businessBookings { slotOptionId status }
    }`,
    {},
    data.ownerToken
  );
  const bookings = (body && body.data && body.data.businessBookings) || [];
  if (bookings.length === 0) {
    console.error(`teardown: businessBookings query returned nothing (status ${res.status}): ${res.body}`);
    return;
  }

  const activeBySlot = {};
  for (const b of bookings) {
    if (b.status === 'cancelled' || b.status === 'rejected') continue;
    activeBySlot[b.slotOptionId] = (activeBySlot[b.slotOptionId] || 0) + 1;
  }

  console.log('\nNFR-1 teardown verification — active bookings per target slot:');
  let allGood = true;
  for (const slotOptionId of data.targetSlots) {
    const count = activeBySlot[slotOptionId] || 0;
    const ok = count === 1;
    if (!ok) allGood = false;
    console.log(`  slot ${slotOptionId}: ${count} active booking(s) ${ok ? 'OK' : '*** VIOLATION ***'}`);
  }
  console.log(allGood ? 'RESULT: PASS — every contended slot ended up with exactly one active booking.' : 'RESULT: FAIL — see violations above.');
}

export function handleSummary(data) {
  const count = (name) => (data.metrics[name] ? data.metrics[name].values.count : 0);
  return {
    stdout: `
NFR-1 concurrency race summary
-------------------------------
Contended slots:            ${SLOTS}
Attempts per slot:          ${ATTEMPTS_PER_SLOT}  (mix of createBooking and rescheduleBooking)
Total concurrent requests:  ${TOTAL_VUS}
Successful landings:        ${count('successful_landings')}  (expected: exactly ${SLOTS} — one per slot)
Rejected as already booked: ${count('already_booked_rejections')}  (expected: ${TOTAL_VUS - SLOTS})
Unexpected errors:          ${count('unexpected_errors')}  (expected: 0)
${count('successful_landings') === SLOTS && count('unexpected_errors') === 0 ? 'CLIENT-SIDE RESULT: PASS' : 'CLIENT-SIDE RESULT: FAIL — see above'}
(see teardown log below for the independent server-side verification)
`,
  };
}
