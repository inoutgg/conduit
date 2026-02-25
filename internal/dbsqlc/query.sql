-- name: AcquireLock :exec
SELECT pg_advisory_lock(@lock_num::BIGINT);

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(@lock_num::BIGINT);

-- name: ResetConn :exec
RESET ALL;

-- name: AllExistingMigrations :many
SELECT version, name
FROM conduit_migrations
ORDER BY version, name;

-- name: ApplyMigration :exec
INSERT INTO conduit_migrations (version, name, hash)
VALUES (@version, @name, @hash);

-- name: RollbackMigration :exec
DELETE FROM conduit_migrations
WHERE version = @version AND name = @name;

-- name: LatestSchemaHash :one
SELECT hash FROM conduit_migrations
ORDER BY version DESC, name DESC
LIMIT 1;

-- name: TestAllMigrations :many
SELECT version, name
FROM conduit_migrations
ORDER BY version, name;

-- name: DoesTableExist :one
SELECT
  CASE
    WHEN to_regclass(@table_name) IS NULL THEN FALSE
    ELSE TRUE
  END;
