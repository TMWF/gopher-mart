package model

type UserRegisterRequestModel struct {
	Login    string `json:"login" validate:"required,min=1"`
	Password string `json:"password" validate:"required,min=1"`
}

type UserLoginRequestModel struct {
	Login    string `json:"login" validate:"required,min=1"`
	Password string `json:"password" validate:"required,min=1"`
}
