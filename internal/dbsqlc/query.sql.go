// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package dbsqlc

import (
	"context"

	"github.com/google/uuid"
)

const acquireLock = `-- name: AcquireLock :exec
SELECT pg_advisory_lock($1::BIGINT)
`

func (q *Queries) AcquireLock(ctx context.Context, db DBTX, lockNum int64) error {
	_, err := db.Exec(ctx, acquireLock, lockNum)
	return err
}

const allExistingMigrationVersions = `-- name: AllExistingMigrationVersions :many
SELECT version
FROM migrations
WHERE namespace = $1
ORDER BY version
`

func (q *Queries) AllExistingMigrationVersions(ctx context.Context, db DBTX, namespace string) ([]int64, error) {
	rows, err := db.Query(ctx, allExistingMigrationVersions, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []int64
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		items = append(items, version)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type ApplyMigrationParams struct {
	ID        uuid.UUID
	Version   int64
	Name      string
	Namespace string
}

const doesTableExist = `-- name: DoesTableExist :one
SELECT COALESCE(to_regclass($1), FALSE) = FALSE
`

func (q *Queries) DoesTableExist(ctx context.Context, db DBTX, tableName string) (bool, error) {
	row := db.QueryRow(ctx, doesTableExist, tableName)
	var column_1 bool
	err := row.Scan(&column_1)
	return column_1, err
}

const releaseLock = `-- name: ReleaseLock :exec
SELECT pg_advisory_unlock($1::BIGINT)
`

func (q *Queries) ReleaseLock(ctx context.Context, db DBTX, lockNum int64) error {
	_, err := db.Exec(ctx, releaseLock, lockNum)
	return err
}

const rollbackMigrations = `-- name: RollbackMigrations :exec
DELETE FROM migrations
WHERE
  (version, namespace) = ANY (
    SELECT unnest($1::BIGINT[]), unnest($2::VARCHAR[])
  )
`

type RollbackMigrationsParams struct {
	Versions   []int64
	Namespaces []string
}

func (q *Queries) RollbackMigrations(ctx context.Context, db DBTX, arg RollbackMigrationsParams) error {
	_, err := db.Exec(ctx, rollbackMigrations, arg.Versions, arg.Namespaces)
	return err
}
