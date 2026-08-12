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
