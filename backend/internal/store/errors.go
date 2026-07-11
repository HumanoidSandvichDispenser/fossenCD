package store

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("already exists")
	ErrLastOwner = errors.New("cannot remove the last owner")
)
