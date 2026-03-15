package service

import "errors"

var (
	ErrUserNotAuthenticated    = errors.New("user not authenticated")
	ErrNotFoundUserWithDrawals = errors.New("not found user withdrawals")
)
