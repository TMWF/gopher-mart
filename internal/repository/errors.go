package repository

import "errors"

var (
	ErrBalanceNotEnough                  = errors.New("not enough bonuses on user balance")
	ErrIncorrectUserOrder                = errors.New("incorrect user order")
	ErrOrderAlreadyUploadedByAnotherUser = errors.New("order already uploaded by another user")
	ErrOrderAlreadyUploadedByThisUser    = errors.New("order already uploaded by this user")
	ErrUserAlreadyExists                 = errors.New("user with such login already exists")
	ErrUserDoesNotExist                  = errors.New("user with such login does not exist")
)
