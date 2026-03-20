package service

import "errors"

// ErrUnauthorized is returned when credentials are invalid.
var ErrUnauthorized = errors.New("unauthorized")

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")
