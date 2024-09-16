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

const findAllExistingMigrations = `-- name: FindAllExistingMigrations :many
SELECT version, name
FROM migrations
WHERE namespace = $1
ORDER BY version
`

type FindAllExistingMigrationsRow struct {
	Version int64
	Name    string
}

func (q *Queries) FindAllExistingMigrations(ctx context.Context, db DBTX, namespace string) ([]FindAllExistingMigrationsRow, error) {
	rows, err := db.Query(ctx, findAllExistingMigrations, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindAllExistingMigrationsRow
	for rows.Next() {
		var i FindAllExistingMigrationsRow
		if err := rows.Scan(&i.Version, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const releaseLock = `-- name: ReleaseLock :exec
SELECT pg_advisory_unlock($1::BIGINT)
`

func (q *Queries) ReleaseLock(ctx context.Context, db DBTX, lockNum int64) error {
	_, err := db.Exec(ctx, releaseLock, lockNum)
	return err
}

const rollbackMigration = `-- name: RollbackMigration :exec
DELETE FROM migrations
WHERE version = $1 AND namespace = $2
`

type RollbackMigrationParams struct {
	Version   int64
	Namespace string
}

func (q *Queries) RollbackMigration(ctx context.Context, db DBTX, arg RollbackMigrationParams) error {
	_, err := db.Exec(ctx, rollbackMigration, arg.Version, arg.Namespace)
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