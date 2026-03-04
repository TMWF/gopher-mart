package model

type UseRegisterRequestModel struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserLoginRequestModel struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
