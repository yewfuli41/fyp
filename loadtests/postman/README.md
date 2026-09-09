# Integration tests (Postman / newman)

`booking-system-integration.postman_collection.json` drives the **GraphQL API** through the
workflows described by the use cases (UC-1 … UC-16). Each folder is one use case; each request is
one numbered integration test (`IT-xx`) that exercises a complete flow across the API, the service
layer and the database, then asserts the resulting state change.

These are **not** unit tests. They do not re-check individual validation rules in isolation — they
follow a workflow end to end (for example: staff applies for leave → owner sees the customer
appointments it would affect → owner approves it with a replacement time → the customer's
appointment is confirmed at that new time).

## Run order matters

Requests must run **top to bottom** — later tests reuse the accounts, business, services, slots and
bookings created earlier. Every run stamps its own data with a unique run id, so repeated runs do
not collide.

## Running it

Prerequisites: the Go server running and reachable, pointed at a **disposable test database** — the
collection creates real accounts, a real business and real bookings.

```bash
# Postman: File > Import > this .json, then Run the collection (top to bottom).

# Or on the command line:
npx newman run loadtests/postman/booking-system-integration.postman_collection.json \
  --env-var baseUrl=http://localhost:8080/query
```

A clean run reports 147 requests / 147 assertions / 0 failures — 93 use-case tests (IT-xx), 35 state transition
steps (ST-xx) and 19 teardown steps (CL-xx).

## Cleanup

The collection's last folder, **`ZZ Cleanup`**, runs only after every test has finished and removes
just what this run created, using the ids captured during the run: its bookings, leave applications,
staff member, appointment slots and service. Nothing is truncated, and it only ever touches records
this run made.

Two things have no delete operation in the API — **accounts and business profiles** — so those are
removed by `cleanup_test_data.sql`, matched on the run's own stamp. The collection prints that stamp
when it finishes:

```
Cleanup done for run stamp 1787843014463. …
```

```bash
psql -h localhost -p 5433 -U postgres -d fyp \
     -v stamp=1787843014463 -f loadtests/postman/cleanup_test_data.sql

# or clear every past run at once, using a SQL wildcard:
psql ... -v stamp=% -f loadtests/postman/cleanup_test_data.sql
```

It matches only emails of the form `it-*-<stamp>@itest.local` (and the business email
`it-biz-<stamp>@itest.local`), so pre-existing data can never match. This was verified by planting an
unrelated account and business before a run: after cleanup, the run's 4 accounts and 1 business were
gone and the planted pair was still there.

## State transition tests

The `Booking State Transitions` folder walks the booking state model itself: every row of the state
transition table (Actor / Current State / Event / Next State), driven through the API. Steps that only
put a booking into the state the next transition needs are marked **(setup)** in the results table.

It also covers the one transition nobody triggers — a booking whose appointment time has gone by moves
to **Past** on its own. That is reached by recording a walk-in for a time earlier today (walk-ins may
be backdated), then reading the bookings, which runs the sweep.

- `state_transition_test_results.md` / `.csv` — results plus an Actor/State/Event/Next State coverage map.

Note: the Past step needs the run to start at least a minute into the day, since the walk-in it uses is
dated today 00:00-00:01.

## Results

- `integration_test_results.md` — the results table (Test ID, Use Case, Test Case, Test Data,
  Expected Result, Actual Result, Status) plus the Use Case → Test ID coverage mapping.
- `integration_test_results.csv` — the same table for pasting into the report.

The "Actual Result" column holds the real response evidence captured from a passing run
(`newman ... --reporters json`), not a placeholder.
