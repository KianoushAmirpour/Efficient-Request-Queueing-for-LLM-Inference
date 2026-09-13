package domain

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserNotAdmin = errors.New("user is not an admin")
)
