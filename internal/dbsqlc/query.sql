-- name: AcquireLock :exec
SELECT pg_advisory_lock(@lock_num::BIGINT);

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(@lock_num::BIGINT);

-- name: ResetConn :exec
RESET ALL;

-- name: AllExistingMigrationVersions :many
SELECT version
FROM conduitmigrations
WHERE namespace = @namespace
ORDER BY version;

-- name: ApplyMigration :exec
INSERT INTO conduitmigrations (version, name, namespace)
VALUES (@version, @name, @namespace);

-- name: RollbackMigration :exec
DELETE FROM conduitmigrations
WHERE version = @version AND namespace = @namespace;

-- name: DoesTableExist :one
SELECT
  CASE
    WHEN to_regclass(@table_name) IS NULL THEN FALSE
    ELSE TRUE
  END;
