package domain

import "errors"

var (
	// ErrNotFound indicates that an entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict indicates a business conflict such as overlapping reservations.
	ErrConflict = errors.New("conflict")
	// ErrUnauthorized indicates missing or invalid authentication.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden indicates that the actor is authenticated but not allowed to perform the action.
	ErrForbidden = errors.New("forbidden")
	// ErrValidation indicates invalid input.
	ErrValidation = errors.New("validation")
)
