REVOKE ALL ON DATABASE postgres_db FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kian_admin') THEN
        CREATE ROLE kian_admin WITH LOGIN CREATEDB CREATEROLE CONNECTION LIMIT 5;
    END IF;

    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'migration_user') THEN
        CREATE ROLE migration_user WITH LOGIN CONNECTION LIMIT 10;
    END IF;

    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user WITH LOGIN;
    END IF;
END
$$;


ALTER SCHEMA public OWNER TO migration_user;

GRANT CONNECT ON DATABASE postgres_db TO kian_admin, migration_user, app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT ALL ON SCHEMA public TO kian_admin;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_user;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO app_user;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT EXECUTE ON FUNCTIONS TO app_user;

-- Give kian_admin Full Control over everything migration_user creates
ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT ALL ON TABLES TO kian_admin;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT ALL ON SEQUENCES TO kian_admin;

ALTER DEFAULT PRIVILEGES FOR ROLE migration_user IN SCHEMA public
    GRANT ALL ON FUNCTIONS TO kian_admin;