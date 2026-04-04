package model

import "github.com/google/uuid"

type AccrualResponseModel struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type OrderModel struct {
	ID     uuid.UUID
	Num    string
	UserID uuid.UUID
	Status string
}
