-- Removes the rows one Postman integration-test run created, and nothing else.
--
-- The collection's own "ZZ Cleanup" folder already undoes everything the API
-- can undo (bookings, leave, staff, slots, service). Two things have no API to
-- remove them at all — the three accounts and the business profile — so they
-- are cleaned up here.
--
-- Everything below is keyed on the run's own stamp, which the collection prints
-- when it finishes ("Cleanup done for run stamp 1724…"). Only rows whose email
-- was generated from that stamp are touched: nothing pre-existing matches those
-- patterns, and no table is ever truncated.
--
-- Usage (psql), passing the stamp the run printed:
--   psql -h localhost -p 5433 -U postgres -d fyp \
--        -v stamp=1724500000000 -f loadtests/postman/cleanup_test_data.sql
--
-- To clear every run at once instead of one, pass  -v stamp=%  (SQL wildcard).

\set ON_ERROR_STOP on

BEGIN;

-- The accounts this run created: it-cust-<stamp>@itest.local,
-- it-cust2-<stamp>@itest.local, it-owner-<stamp>@itest.local and
-- it-staff-<stamp>@itest.local. The suffix is what keeps real accounts out.
CREATE TEMP TABLE run_users ON COMMIT DROP AS
SELECT user_id, email
FROM fyp_fuli_users
WHERE email LIKE 'it-cust-'  || :'stamp' || '@itest.local'
   OR email LIKE 'it-cust2-' || :'stamp' || '@itest.local'
   OR email LIKE 'it-owner-' || :'stamp' || '@itest.local'
   OR email LIKE 'it-staff-' || :'stamp' || '@itest.local';

CREATE TEMP TABLE run_businesses ON COMMIT DROP AS
SELECT business_id
FROM fyp_fuli_business_profiles
WHERE owner_user_id IN (SELECT user_id FROM run_users)
   OR business_email LIKE 'it-biz-' || :'stamp' || '@itest.local';

\echo 'Rows matched for this stamp:'
SELECT (SELECT count(*) FROM run_users)      AS accounts,
       (SELECT count(*) FROM run_businesses) AS businesses;

-- Children first: each delete is scoped to the run's own business or accounts.
DELETE FROM fyp_fuli_bookings b
USING fyp_fuli_service_slot_options sso, fyp_fuli_service_slots ss, fyp_fuli_staff st
WHERE b.slot_option_id = sso.slot_option_id
  AND sso.service_slot_id = ss.service_slot_id
  AND ss.staff_id = st.staff_id
  AND st.business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_bookings b
USING fyp_fuli_service_slot_options sso, fyp_fuli_service_options so, fyp_fuli_services sv
WHERE b.slot_option_id = sso.slot_option_id
  AND sso.service_option_id = so.service_option_id
  AND so.service_id = sv.service_id
  AND sv.business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_bookings
WHERE user_id IN (SELECT user_id FROM run_users);

DELETE FROM fyp_fuli_service_slot_options sso
USING fyp_fuli_service_options so, fyp_fuli_services sv
WHERE sso.service_option_id = so.service_option_id
  AND so.service_id = sv.service_id
  AND sv.business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_service_slots ss
USING fyp_fuli_staff st
WHERE ss.staff_id = st.staff_id
  AND st.business_id IN (SELECT business_id FROM run_businesses);

-- Owner-managed slots carry no staff id, so they are reached through the
-- recurring schedule or through the creator instead.
DELETE FROM fyp_fuli_service_slots
WHERE recurring_schedule_id IN (
        SELECT recurring_schedule_id FROM fyp_fuli_recurring_schedules
        WHERE business_id IN (SELECT business_id FROM run_businesses))
   OR created_by IN (SELECT user_id FROM run_users);

DELETE FROM fyp_fuli_recurring_schedules
WHERE business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_leave_applications
WHERE staff_id IN (SELECT staff_id FROM fyp_fuli_staff
                   WHERE business_id IN (SELECT business_id FROM run_businesses));

DELETE FROM fyp_fuli_staff_working_hours
WHERE staff_id IN (SELECT staff_id FROM fyp_fuli_staff
                   WHERE business_id IN (SELECT business_id FROM run_businesses));

DELETE FROM fyp_fuli_staff
WHERE business_id IN (SELECT business_id FROM run_businesses)
   OR user_id IN (SELECT user_id FROM run_users);

DELETE FROM fyp_fuli_service_option_items
WHERE service_option_id IN (
        SELECT so.service_option_id FROM fyp_fuli_service_options so
        JOIN fyp_fuli_services sv ON sv.service_id = so.service_id
        WHERE sv.business_id IN (SELECT business_id FROM run_businesses));

DELETE FROM fyp_fuli_service_options
WHERE service_id IN (SELECT service_id FROM fyp_fuli_services
                     WHERE business_id IN (SELECT business_id FROM run_businesses));

DELETE FROM fyp_fuli_services
WHERE business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_business_working_hours
WHERE business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_business_profiles
WHERE business_id IN (SELECT business_id FROM run_businesses);

DELETE FROM fyp_fuli_users
WHERE user_id IN (SELECT user_id FROM run_users);

\echo 'Left behind for this stamp (all should be 0):'
SELECT (SELECT count(*) FROM fyp_fuli_users
        WHERE email LIKE '%' || :'stamp' || '@itest.local')            AS accounts_left,
       (SELECT count(*) FROM fyp_fuli_business_profiles
        WHERE business_email LIKE '%' || :'stamp' || '@itest.local')   AS businesses_left;

COMMIT;
