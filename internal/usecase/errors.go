package usecase

import "errors"

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotFound      = errors.New("not found")
	ErrLimitExceeded = errors.New("limit exceeded")
	ErrConflict      = errors.New("conflict")
)
