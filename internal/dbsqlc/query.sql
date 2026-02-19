-- name: AcquireLock :exec
SELECT pg_advisory_lock(@lock_num::BIGINT);

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(@lock_num::BIGINT);

-- name: ResetConn :exec
RESET ALL;

-- name: AllExistingMigrationVersions :many
SELECT version
FROM conduit_migrations
ORDER BY version;

-- name: ApplyMigration :exec
INSERT INTO conduit_migrations (version, name, hash)
VALUES (@version, @name, @hash);

-- name: RollbackMigration :exec
DELETE FROM conduit_migrations
WHERE version = @version;

-- name: LatestSchemaHash :one
SELECT hash FROM conduit_migrations
ORDER BY version DESC
LIMIT 1;

-- name: DoesTableExist :one
SELECT
  CASE
    WHEN to_regclass(@table_name) IS NULL THEN FALSE
    ELSE TRUE
  END;
