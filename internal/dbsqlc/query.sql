-- name: AcquireLock :exec
SELECT pg_advisory_lock(@lock_num::BIGINT);

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(@lock_num::BIGINT);

-- name: ResetConn :exec
RESET ALL;

-- name: AllExistingMigrationVersions :many
SELECT version
FROM conduit_migrations
WHERE namespace = @namespace
ORDER BY version;

-- name: ApplyMigration :exec
INSERT INTO  conduit_migrations (version, name, namespace)
VALUES (@version, @name, @namespace);

-- name: RollbackMigration :exec
DELETE FROM conduit_migrations
WHERE version = @version AND namespace = @namespace;

-- name: DoesTableExist :one
SELECT
  CASE
    WHEN to_regclass(@table_name) IS NULL THEN FALSE
    ELSE TRUE
  END;
