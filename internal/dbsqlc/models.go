// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package dbsqlc

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Conduitmigration struct {
	ID        uuid.UUID
	CreatedAt pgtype.Timestamp
	Version   int64
	Name      string
	Namespace string
}
