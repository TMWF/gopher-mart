package model

import "time"

type GetBalanceResponseModel struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithDrawBalanceRequestModel struct {
	Order string  `json:"order" validate:"required,min=1,numeric,luhn"`
	Sum   float64 `json:"sum" validate:"required,min=1"`
}

type GetUserWithdrawalsResponseModel struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
