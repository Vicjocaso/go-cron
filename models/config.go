package models

import "time"

type AppConfig struct {
	ServerPort   uint16
	Database     DatabaseConfig
	Auth         AuthConfig
	ExternalAuth ExternalAuthConfig
	ExternalAPI  ExternalApiConfig
}

type DatabaseConfig struct {
	DatabaseURI     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	DataSourceURL   string
}

type AuthConfig struct {
	CRONSecret string
}

type ExternalAuthConfig struct {
	CompanyDB string `json:"CompanyDB"`
	UserName  string `json:"UserName"`
	Password  string `json:"Password"`
}

type ExternalApiConfig struct {
	ExternalAPIURL string
	LoginURL       string
	ItemsURL       string
	Filter         string
}
