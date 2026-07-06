package storage

import (
	"errors"
	"fmt"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const pgUniqueViolation = "23505"

// MapError переводит ошибки pgx в доменные:
// pgx.ErrNoRows -> apperr.ErrNotFound, unique violation -> apperr.ErrConflict.
// Остальные ошибки оборачиваются с контекстом операции (внутренняя ошибка -> 500).
func MapError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return apperr.ErrConflict
	}
	return fmt.Errorf("%s: %w", op, err)
}
