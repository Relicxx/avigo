// Package apperr содержит доменные ошибки-сентинелы, единые для всех слоёв.
// Репозитории маппят ошибки БД на них, хендлеры — на HTTP-коды.
package apperr

import "errors"

var (
	// ErrNotFound — сущность не найдена (или не принадлежит пользователю).
	ErrNotFound = errors.New("not found")
	// ErrConflict — нарушение уникальности или конфликт состояния.
	ErrConflict = errors.New("conflict")
	// ErrForbidden — действие запрещено для данного пользователя.
	ErrForbidden = errors.New("forbidden")
	// ErrUnauthorized — не аутентифицирован или невалидный токен.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidInput — некорректные данные запроса.
	ErrInvalidInput = errors.New("invalid input")
)
