package model

import (
	"fmt"

	"github.com/google/uuid"
)

type AccrualResponseModel struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

func (a AccrualResponseModel) String() string {
	return fmt.Sprintf("AccrualResponse(order=%s, status=%s, accrual=%.2f)",
		a.Order, a.Status, a.Accrual)
}

type OrderModel struct {
	ID     uuid.UUID
	Num    string
	UserID uuid.UUID
	Status string
}

func (om *OrderModel) String() string {
	return "id: " + om.ID.String() + ", " + "num: " + om.Num + ", " + "userID: " + om.UserID.String() + ", " + "status: " + om.Status
}
