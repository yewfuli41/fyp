DO $$
BEGIN
    IF NOT EXISTS (
        SELECT
        FROM pg_roles
        WHERE rolname =
            'booking_app_user'
    ) THEN

        CREATE ROLE
            booking_app_user
        LOGIN PASSWORD
            'password';

    END IF;
END
$$;

GRANT
SELECT,
INSERT,
UPDATE,
DELETE
ON ALL TABLES
IN SCHEMA public
TO booking_app_user;