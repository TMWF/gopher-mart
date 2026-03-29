package service

import "errors"

var (
	ErrUserNotAuthenticated    = errors.New("user not authenticated")
	ErrNotFoundUserWithDrawals = errors.New("not found user withdrawals")
	ErrNotFoundUserOrders      = errors.New("not found user orders")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrUserDoesNotExist        = errors.New("user does not exist")
)
