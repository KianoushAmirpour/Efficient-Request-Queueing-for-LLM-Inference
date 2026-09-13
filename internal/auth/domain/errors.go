package domain

import "errors"

var (
	ErrOauthAccountNotFound   = errors.New("oauth account not found")
	ErrStateParameterNotFound = errors.New("state parameter not found")
)
