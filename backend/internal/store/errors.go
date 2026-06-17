package store

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrLastOwner = errors.New("cannot remove the last owner")
)
