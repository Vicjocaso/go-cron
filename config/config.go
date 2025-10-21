package config

import (
	"go-cron/models"
	"os"
	"time"
)

func LoadConfig() *models.AppConfig {
	cfg := &models.AppConfig{
		ServerPort: 3000,
		Database: models.DatabaseConfig{
			DatabaseURI:     os.Getenv("DATABASE_URL"),
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 10 * time.Minute,
		},
		Auth: models.AuthConfig{
			CRONSecret: os.Getenv("CRON_SECRET"),
		},
		ExternalAPI: models.ExternalApiConfig{
			LoginURL:       "/Login",
			ItemsURL:       "/Items",
			ExternalAPIURL: os.Getenv("EXTERNAL_API_URL"),
			Filter:         "?$select=ItemCode,ItemName,ItemsGroupCode&$filter=ItemsGroupCode eq 100 or ItemsGroupCode eq 101 or ItemsGroupCode eq 121 or ItemsGroupCode eq 118&$orderby=ItemCode",
		},
		ExternalAuth: models.ExternalAuthConfig{
			CompanyDB: os.Getenv("COMPANY_DB"),
			UserName:  os.Getenv("USER_NAME"),
			Password:  os.Getenv("PASSWORD"),
		},
	}
	return cfg
}
