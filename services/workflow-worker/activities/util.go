package activities

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}
