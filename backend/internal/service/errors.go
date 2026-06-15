package service

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrConflict           = errors.New("already exists")
	ErrNotFound           = errors.New("not found")
)
