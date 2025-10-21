package models

type Credentials struct {
	CompanyDB string `json:"CompanyDB"`
	UserName  string `json:"UserName"`
	Password  string `json:"Password"`
}

type LoginResponse struct {
	SessionID      string `json:"SessionId"`
	Version        string `json:"Version"`
	SessionTimeout int    `json:"SessionTimeout"`
}
