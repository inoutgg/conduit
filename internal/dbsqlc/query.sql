-- name: AcquireLock :exec
SELECT pg_advisory_lock(@lock_num::BIGINT);

-- name: ReleaseLock :exec
SELECT pg_advisory_unlock(@lock_num::BIGINT);

-- name: FindAllExistingMigrations :many
SELECT version, name
FROM migrations
WHERE namespace = @namespace
ORDER BY version;

-- name: ApplyMigration :copyfrom
INSERT INTO migrations (id, version, name, namespace)
VALUES (@id, @version, @name, @namespace);

-- name: RollbackMigrations :exec
DELETE FROM migrations
WHERE
  (version, namespace) = ANY (
    SELECT unnest(@versions::BIGINT[]), unnest(@namespaces::VARCHAR[])
  );

-- name: DoesTableExist :one
SELECT COALESCE(to_regclass(@table_name), FALSE) = FALSE;
