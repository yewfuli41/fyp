// Leave-approval atomicity race test.
//
// Approving a leave with replacement times is meant to be all-or-nothing:
// every affected booking moves onto the replacement slot the owner picked and
// the leave becomes approved together, or none of it happens and the leave
// stays pending — never a half-settled leave with some customers already
// moved (see ApproveLeaveApplication in internal/service/leaveService.go).
//
// The unit tests can only prove that sequentially, against mocks. The way
// this guarantee actually breaks in production is a race: while the owner is
// approving, an ordinary customer books one of the very slots the owner chose
// as a replacement. This script creates that race for real — one owner
// approval against RACERS_PER_SLOT customers per replacement slot, all fired
// at the same instant — and then checks the stored data afterwards.
//
// Whoever loses must lose cleanly:
//   - approval wins  -> leave APPROVED and EVERY affected booking sits on its
//                       replacement slot; the racing customers are rejected.
//   - a racer wins   -> approval refused ("just taken" / "no longer
//                       available"), leave still PENDING, and EVERY affected
//                       booking still sits on its ORIGINAL slot. One booking
//                       moved while the leave stayed pending is the failure
//                       this test exists to catch.
//
// Usage:
//   k6 run leave_approval_race.js
//   AFFECTED=3 RACERS_PER_SLOT=15 k6 run leave_approval_race.js
//
// Prerequisites: same as the other scripts here — a running server pointed at
// a disposable/test database. This creates real accounts, a business, a staff
// member, slots, bookings and a leave application on every run.

import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

// Dates must be worked out in LOCAL time, not UTC. The server compares them
// against its own clock, so before 08:00 in a UTC+8 timezone the UTC date is
// still yesterday — which made "today" land in the past and the run fail in
// setup. toISOString() is UTC, so shift by the timezone offset first.
function localDate(offsetDays) {
  const d = new Date(Date.now() + offsetDays * 86400000);
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
}

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/query';
// One affected booking per replacement slot; the owner must settle all of
// them in the same approval, which is exactly what makes this atomic.
const AFFECTED = Number(__ENV.AFFECTED || 2);
const RACERS_PER_SLOT = Number(__ENV.RACERS_PER_SLOT || 10);
const TOTAL_VUS = 1 + AFFECTED * RACERS_PER_SLOT; // VU 1 approves, the rest race it

const approvalWon = new Counter('approval_won');
const approvalCleanlyRefused = new Counter('approval_cleanly_refused');
const racerWins = new Counter('racer_wins');
const racerRejections = new Counter('racer_rejections');
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
    // The approval must always end in one of the two clean outcomes — never
    // a crash, and never a partially applied approval.
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

function dig(body, ...path) {
  let node = body && body.data;
  for (const key of path) {
    if (node === null || node === undefined) return undefined;
    node = node[key];
  }
  return node;
}

function must(result, value, label) {
  if (value === null || value === undefined) {
    throw new Error(`${label} failed: ${result.res.status} ${result.res.body}`);
  }
  return value;
}

const SIGN_UP = `mutation($u: SignUpInput!) { signUp(user: $u) { token } }`;
const LOG_IN = `mutation($u: LogInInput!) { logIn(user: $u) { token } }`;

function signUp(stamp, label) {
  const r = gql(SIGN_UP, {
    u: {
      username: `K6${label}${stamp}`,
      email: `${label}-${stamp}@k6test.local`,
      contactNumber: '0123456789',
      password: 'Password123',
    },
  });
  return must(r, dig(r.body, 'signUp', 'token'), `${label} signUp`);
}

function dateOffset(days) {
  return localDate(days);
}

// setup() runs once, single-threaded, before any VU starts. It builds the
// exact situation an owner faces when approving a leave that has bookings on
// it, and stops right before the approval itself.
export function setup() {
  const stamp = Date.now();
  const ownerToken = signUp(stamp, 'owner');

  const allDayHours = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'].map((day) => ({
    day,
    startTime: '2000-01-01T00:00:00Z',
    endTime: '2000-01-01T23:59:00Z',
  }));

  const biz = gql(
    `mutation($b: BusinessProfileInput!) { registerBusinessProfile(business: $b) { businessId } }`,
    {
      b: {
        businessName: `K6 Leave Biz ${stamp}`,
        address: '1 Test Street',
        businessContactNumber: '0123456789',
        businessEmail: `biz-${stamp}@k6test.local`,
        workingHours: allDayHours,
      },
    },
    ownerToken
  );
  must(biz, dig(biz.body, 'registerBusinessProfile', 'businessId'), 'registerBusinessProfile');

  const service = gql(
    `mutation($s: ServiceInput!) { createService(service: $s) { serviceOptions { serviceOptionId } } }`,
    {
      s: {
        serviceName: `K6 Leave Service ${stamp}`,
        serviceOptions: [
          {
            serviceOptionName: 'Standard',
            serviceOptionItems: [{ serviceOptionItemName: 'Standard item' }],
            effectiveFrom: dateOffset(0),
          },
        ],
      },
    },
    ownerToken
  );
  const serviceOptionId = must(
    service,
    dig(service.body, 'createService', 'serviceOptions', 0, 'serviceOptionId'),
    'createService'
  );

  // The staff member who will take the leave. registerStaff creates the
  // account with a temporary password; the staff has to reset it before the
  // account is usable, exactly as a real staff member's first login does.
  const staffEmail = `staff-${stamp}@k6test.local`;
  const staff = gql(
    `mutation($s: StaffInput!) { registerStaff(staff: $s) }`,
    {
      s: {
        name: `K6 Staff ${stamp}`,
        email: staffEmail,
        contactNumber: '0123456789',
        position: 'Technician',
        password: 'TempPass123',
        workingHours: allDayHours,
      },
    },
    ownerToken
  );
  must(staff, dig(staff.body, 'registerStaff'), 'registerStaff');

  const firstLogin = gql(LOG_IN, { u: { email: staffEmail, password: 'TempPass123' } });
  const firstToken = must(firstLogin, dig(firstLogin.body, 'logIn', 'token'), 'staff first logIn');
  const reset = gql(`mutation($p: String!) { resetPassword(newPassword: $p) }`, { p: 'Password123' }, firstToken);
  must(reset, dig(reset.body, 'resetPassword'), 'staff resetPassword');
  const relogin = gql(LOG_IN, { u: { email: staffEmail, password: 'Password123' } });
  const staffToken = must(relogin, dig(relogin.body, 'logIn', 'token'), 'staff logIn');

  const staffList = gql(`query { displayStaff { staffId email } }`, {}, ownerToken);
  const staffRow = must(
    staffList,
    (dig(staffList.body, 'displayStaff') || []).filter((s) => s.email === staffEmail)[0],
    'displayStaff'
  );
  const staffId = staffRow.staffId;

  function createSlot(date, hour, assignToStaff) {
    const hh = String(hour).padStart(2, '0');
    const input = {
      date,
      startTime: `2000-01-01T${hh}:00:00Z`,
      endTime: `2000-01-01T${hh}:59:00Z`,
      serviceOptionIds: [serviceOptionId],
    };
    if (assignToStaff) input.staffId = staffId;
    const slot = gql(
      `mutation($i: ServiceSlotInput!) { createServiceSlot(input: $i) { serviceSlotId serviceSlotOptions { slotOptionId } } }`,
      { i: input },
      ownerToken
    );
    return {
      serviceSlotId: must(slot, dig(slot.body, 'createServiceSlot', 'serviceSlotId'), 'createServiceSlot'),
      slotOptionId: must(
        slot,
        dig(slot.body, 'createServiceSlot', 'serviceSlotOptions', 0, 'slotOptionId'),
        'createServiceSlot option'
      ),
    };
  }

  // The leave covers a single future day. The staff member's own slots that
  // day are the ones customers are booked into; the replacement slots are
  // owner-managed, so picking them is allowed (a replacement on the staff's
  // own slot inside their own leave is rejected outright by the service).
  const leaveDate = dateOffset(3);

  const affected = [];
  for (let i = 0; i < AFFECTED; i++) {
    const originSlot = createSlot(leaveDate, 8 + i, true);
    const replacementSlot = createSlot(leaveDate, 14 + i, false);
    const customerToken = signUp(stamp, `cust${i}`);
    const booking = gql(
      `mutation($slotOptionId: ID!) { createBooking(slotOptionId: $slotOptionId) { bookingId } }`,
      { slotOptionId: originSlot.slotOptionId },
      customerToken
    );
    affected.push({
      bookingId: must(booking, dig(booking.body, 'createBooking', 'bookingId'), `affected createBooking ${i}`),
      originSlotOptionId: originSlot.slotOptionId,
      replacementSlotOptionId: replacementSlot.slotOptionId,
    });
  }

  const leave = gql(
    `mutation($i: ApplyLeaveInput!) { applyLeave(input: $i) { leaveId status } }`,
    { i: { startDate: leaveDate, endDate: leaveDate, justification: 'k6 concurrency test leave' } },
    staffToken
  );
  const leaveId = must(leave, dig(leave.body, 'applyLeave', 'leaveId'), 'applyLeave');

  // One racing customer account per contender, signed up ahead of the race so
  // the race itself is nothing but the competing writes.
  const racers = [];
  for (let i = 0; i < AFFECTED * RACERS_PER_SLOT; i++) {
    racers.push({
      token: signUp(stamp, `racer${i}`),
      slotOptionId: affected[i % AFFECTED].replacementSlotOptionId,
    });
  }

  return { ownerToken, leaveId, affected, racers };
}

const CLEAN_REFUSAL = /just taken|no longer available|already been booked/i;

// Validation errors from this API are wrapped: the top-level message is the
// generic "validation failed", with the real per-field message(s) nested in
// extensions.validationErrors[] (see graph/graphErrs/validation_error.go).
function refusedCleanly(body) {
  if (!body || !body.errors) return false;
  return body.errors.some((e) => {
    if (CLEAN_REFUSAL.test(e.message)) return true;
    const ve = e.extensions && e.extensions.validationErrors;
    return Array.isArray(ve) && ve.some((v) => CLEAN_REFUSAL.test(v.message));
  });
}

// default(): VU 1 is the owner approving the leave (settling every affected
// booking in one call); every other VU is a customer booking one of the
// replacement slots out from under that approval, at the same instant.
export default function (data) {
  if (__VU === 1) {
    const reschedules = data.affected.map((a) => ({
      bookingId: a.bookingId,
      newSlotOptionId: a.replacementSlotOptionId,
    }));
    const { res, body } = gql(
      `mutation($leaveId: ID!, $reschedules: [LeaveRescheduleInput!]) {
        approveLeaveApplication(leaveId: $leaveId, reschedules: $reschedules) { leaveId status }
      }`,
      { leaveId: data.leaveId, reschedules },
      data.ownerToken
    );

    const approved = dig(body, 'approveLeaveApplication', 'status') === 'APPROVED';
    const refused = refusedCleanly(body);
    check(res, { 'approval either went through or was cleanly refused': () => approved || refused });

    if (approved) {
      approvalWon.add(1);
    } else if (refused) {
      approvalCleanlyRefused.add(1);
    } else {
      unexpectedErrors.add(1);
      console.error(`approval got an unexpected response (status ${res.status}): ${res.body}`);
    }
    return;
  }

  const racer = data.racers[__VU - 2];
  const { res, body } = gql(
    `mutation($slotOptionId: ID!) { createBooking(slotOptionId: $slotOptionId) { bookingId } }`,
    { slotOptionId: racer.slotOptionId },
    racer.token
  );

  const booked = !!dig(body, 'createBooking', 'bookingId');
  const rejected = refusedCleanly(body);
  check(res, { 'racer either booked or was cleanly rejected as unavailable': () => booked || rejected });

  if (booked) {
    racerWins.add(1);
  } else if (rejected) {
    racerRejections.add(1);
  } else {
    unexpectedErrors.add(1);
    console.error(`racer VU ${__VU} got an unexpected response (status ${res.status}): ${res.body}`);
  }
}

// teardown() is the real assertion: read the stored state back and check that
// the leave and every one of its bookings agree with each other. A leave that
// ended up PENDING with a booking already moved — or APPROVED with a booking
// left behind — is the all-or-nothing violation this test looks for.
export function teardown(data) {
  const leaves = gql(`query { businessLeaveApplications { leaveId status } }`, {}, data.ownerToken);
  const leave = (dig(leaves.body, 'businessLeaveApplications') || []).filter(
    (l) => String(l.leaveId) === String(data.leaveId)
  )[0];
  if (!leave) {
    console.error(`teardown: leave ${data.leaveId} not found (status ${leaves.res.status}): ${leaves.res.body}`);
    return;
  }

  const bookings = gql(`query { businessBookings { bookingId slotOptionId status } }`, {}, data.ownerToken);
  const byId = {};
  for (const b of dig(bookings.body, 'businessBookings') || []) byId[String(b.bookingId)] = b;

  console.log(`\nLeave-approval race verification — leave ended up ${leave.status}:`);
  const expectMoved = leave.status === 'APPROVED';
  let allGood = true;

  for (const a of data.affected) {
    const b = byId[String(a.bookingId)];
    if (!b) {
      console.log(`  booking ${a.bookingId}: MISSING from businessBookings *** VIOLATION ***`);
      allGood = false;
      continue;
    }
    const moved = String(b.slotOptionId) === String(a.replacementSlotOptionId);
    const stayed = String(b.slotOptionId) === String(a.originSlotOptionId);
    const ok = expectMoved ? moved : stayed;
    if (!ok) allGood = false;
    console.log(
      `  booking ${a.bookingId}: on slot option ${b.slotOptionId} ` +
        `(${moved ? 'replacement' : stayed ? 'original' : 'neither!'}) ${ok ? 'OK' : '*** VIOLATION ***'}`
    );
  }

  // The replacement slots must also still hold at most one active booking
  // each — the approval and a racing customer must never both land.
  const activeBySlot = {};
  for (const b of dig(bookings.body, 'businessBookings') || []) {
    const status = String(b.status).toUpperCase();
    if (status === 'CANCELLED' || status === 'REJECTED') continue;
    activeBySlot[String(b.slotOptionId)] = (activeBySlot[String(b.slotOptionId)] || 0) + 1;
  }
  for (const a of data.affected) {
    const count = activeBySlot[String(a.replacementSlotOptionId)] || 0;
    const ok = count <= 1;
    if (!ok) allGood = false;
    console.log(
      `  replacement slot option ${a.replacementSlotOptionId}: ${count} active booking(s) ${ok ? 'OK' : '*** VIOLATION ***'}`
    );
  }

  console.log(
    allGood
      ? expectMoved
        ? 'RESULT: PASS — the approval won and every affected booking moved with it.'
        : 'RESULT: PASS — the approval lost the race and nothing was moved; the leave is still pending.'
      : 'RESULT: FAIL — the approval was applied only partly; see violations above.'
  );
}

export function handleSummary(data) {
  const count = (name) => (data.metrics[name] ? data.metrics[name].values.count : 0);
  const outcome = count('approval_won') === 1 ? 'the approval won' : 'a racing customer won';
  return {
    stdout: `
Leave-approval atomicity race summary
--------------------------------------
Affected bookings settled in one approval: ${AFFECTED}
Racing customers per replacement slot:     ${RACERS_PER_SLOT}
Total concurrent requests:                 ${TOTAL_VUS}  (1 approval + ${AFFECTED * RACERS_PER_SLOT} bookings)
Approval went through:                     ${count('approval_won')}
Approval cleanly refused:                  ${count('approval_cleanly_refused')}
Racing bookings that landed:               ${count('racer_wins')}
Racing bookings cleanly rejected:          ${count('racer_rejections')}
Unexpected errors:                         ${count('unexpected_errors')}  (expected: 0)
${count('unexpected_errors') === 0 ? `CLIENT-SIDE RESULT: PASS — ${outcome}, cleanly.` : 'CLIENT-SIDE RESULT: FAIL — see above'}
(see teardown log below for the independent all-or-nothing verification)
`,
  };
}
