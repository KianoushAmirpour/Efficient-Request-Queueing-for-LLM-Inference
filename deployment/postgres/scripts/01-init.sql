
-- Remove privileges granted to PUBLIC.
REVOKE ALL ON DATABASE postgres_db FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;

-- Create application roles.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT FROM pg_roles WHERE rolname = 'kian_admin'
    ) THEN
        CREATE ROLE kian_admin WITH LOGIN CONNECTION LIMIT 5;
    END IF;

    IF NOT EXISTS (
        SELECT FROM pg_roles WHERE rolname = 'migration_user'
    ) THEN
        CREATE ROLE migration_user WITH LOGIN CONNECTION LIMIT 10;
    END IF;

    IF NOT EXISTS (
        SELECT FROM pg_roles WHERE rolname = 'app_user'
    ) THEN
        CREATE ROLE app_user WITH LOGIN;
    END IF;
END
$$;

-- Database permissions.
GRANT CONNECT ON DATABASE postgres_db
TO kian_admin, migration_user, app_user;

-- Migration role owns the application schema.
ALTER SCHEMA public OWNER TO migration_user;

-- Allow application access.
GRANT USAGE ON SCHEMA public TO app_user;

-- Allow the administrative account to assume migration privileges.
GRANT migration_user TO kian_admin;

-- Existing objects.
GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA public TO app_user;

GRANT USAGE, SELECT
ON ALL SEQUENCES IN SCHEMA public TO app_user;

GRANT EXECUTE
ON ALL FUNCTIONS IN SCHEMA public TO app_user;

-- Future objects created by migration_user.
ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO app_user;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO app_user;
