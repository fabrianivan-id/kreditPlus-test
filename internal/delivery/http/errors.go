package http

import "errors"

var (
	errInternal       = errors.New("internal server error")
	errUnauthorized   = errors.New("unauthorized")
	errRateLimited    = errors.New("rate limit exceeded")
	errNotFound       = errors.New("not found")
	errMethodNotAllowed = errors.New("method not allowed")
)
