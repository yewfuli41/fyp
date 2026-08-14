# Double-booking race test (k6)

This is a **concurrency test**, not a unit test — it fires many real, concurrent HTTP
requests at a running instance of the app to prove the double-booking guard holds up
under an actual race, not just in a mocked/sequential test. It belongs in a
"Concurrency / Load Testing" section of the report, separate from the unit testing
results table (which excludes integration/system/API tests by design).

## What it proves

`internal/service/booking_service_test.go` already has a unit test proving that
*accepting* a booking rejects other pending requests for the same slot — but that's a
mocked, single-threaded test. It cannot prove what happens when two customers hit
`createBooking` for the **same slot at the same instant**. That race is actually
guarded by the `uq_active_booking_per_slot` unique index in `database/schema.sql`
(with the app catching the resulting DB error and turning it into a clean "this time
slot has already been booked" response — see `CreateBooking` in
`internal/service/bookingService.go`). This script is what exercises that guarantee
for real.

## What it does

1. **Setup** (once, before the race): creates a fresh business owner, business,
   service with one option, and exactly one bookable slot for tomorrow (the single
   "seat" everyone will race for) — plus `VUS` distinct, freshly signed-up customer
   accounts, one per virtual user.
2. **Race**: all `VUS` customers call `createBooking` against that same slot at
   (as close to) the same instant as k6 can manage.
3. **Assertion**: exactly **1** booking should succeed; every other request should be
   cleanly rejected with "no longer available" / "already been booked" — never a
   crash, a duplicate booking, or an unexpected error.

## Prerequisites

- k6 installed (`brew install k6` on macOS).
- The Go server running and reachable (default `http://localhost:8080/query`).
- **A disposable/test database.** This script creates real accounts, a real business,
  and a real slot on every run — never point it at a shared or production database.

## Running it

```bash
# from the repo root, with the server already running against a test DB
k6 run loadtests/k6/double_booking_race.js

# customize the number of racing customers or target URL
VUS=25 BASE_URL=http://localhost:8080/query k6 run loadtests/k6/double_booking_race.js
```

A passing run prints a summary like:

```
Double-booking race test summary
---------------------------------
Concurrent customers (VUs): 10
Successful bookings:        1  (expected: exactly 1)
Rejected as already booked: 9  (expected: 9)
Unexpected errors:          0  (expected: 0)
RESULT: PASS — the slot was never double-booked.
```

---

# NFR-1 concurrency test — `nfr1_concurrency_race.js`

This is the fuller test for **NFR-1**: *"The system shall handle at least 50 concurrent
booking and rescheduling requests while maintaining data consistency, ensuring that no
appointment slot is assigned to more than one active booking."*

`double_booking_race.js` above proves the guarantee for the simplest possible case — one
slot, one operation type. This script exercises the actual requirement:

- **Multiple contended slots at once** — 5 by default (`SLOTS`), not just one.
- **A mix of two different operations** racing for the same slot: `createBooking` (a
  brand-new customer booking it directly) and `rescheduleBooking` (a customer moving an
  *existing* booking from elsewhere onto it). Reschedule requests hit the same
  `uq_active_booking_per_slot` constraint via `UpdateBookingSlotOption`, so this proves
  the guarantee holds across both write paths, not just one.
- **50+ concurrent requests total** — `SLOTS × ATTEMPTS_PER_SLOT` (default 5 × 10 = 50),
  evenly split between the two operation types per slot.
- **An independent server-side check**, not just counting HTTP responses. After the
  race, `teardown()` logs into the business owner's account and queries
  `businessBookings` directly, then verifies for *every* contended slot that exactly one
  non-cancelled/non-rejected booking actually landed there — the real thing NFR-1 is
  asserting, checked against real stored data rather than trusting client-side tallies.

## Running it

```bash
# default: 5 slots x 10 attempts = 50 concurrent requests
k6 run loadtests/k6/nfr1_concurrency_race.js

# scale it up — 8 slots x 10 attempts = 80 concurrent requests
SLOTS=8 ATTEMPTS_PER_SLOT=10 k6 run loadtests/k6/nfr1_concurrency_race.js
```

Same prerequisites as above — a running server pointed at a disposable/test database.

A passing run prints a client-side summary plus the independent teardown check:

```
NFR-1 concurrency race summary
-------------------------------
Contended slots:            5
Attempts per slot:          10  (mix of createBooking and rescheduleBooking)
Total concurrent requests:  50
Successful landings:        5  (expected: exactly 5 — one per slot)
Rejected as already booked: 45  (expected: 45)
Unexpected errors:          0  (expected: 0)
CLIENT-SIDE RESULT: PASS

NFR-1 teardown verification — active bookings per target slot:
  slot 1234: 1 active booking(s) OK
  slot 1235: 1 active booking(s) OK
  slot 1236: 1 active booking(s) OK
  slot 1237: 1 active booking(s) OK
  slot 1238: 1 active booking(s) OK
RESULT: PASS — every contended slot ended up with exactly one active booking.
```
