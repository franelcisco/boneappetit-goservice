package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port  string
	Debug string

	// DATABASE
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	SSLMode    string

	// Shopify API credentials
	ShopifyAPIVersion string
	ShopifyAdminToken string
	ShopifyStoreName  string
	ShopifyHMACSecret string

	// Mailgun API credentials
	MailgunDomain string
	MailgunAPIKey string
	MailgunSender string

	// food-api-dev
	FoodAPIURL     string
	FoodAPITimeout time.Duration
}

// Load reads configuration from environment variables and returns a Config struct
func Load() (*Config, error) {
	foodTimeout := 130 * time.Second

	if timeout, err := strconv.Atoi(os.Getenv("FOOD_API_TIMEOUT")); err == nil {
		foodTimeout = time.Duration(timeout) * time.Second
	}

	cfg := &Config{
		Port:  os.Getenv("PORT"),
		Debug: os.Getenv("DEBUG"),
		// DATABASE
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		SSLMode:    os.Getenv("SSL_MODE"),

		ShopifyAPIVersion: os.Getenv("SHOPIFY_API_VERSION"),
		ShopifyAdminToken: os.Getenv("SHOPIFY_ADMIN_TOKEN"),
		ShopifyStoreName:  os.Getenv("SHOPIFY_STORE_NAME"),
		ShopifyHMACSecret: os.Getenv("SHOPIFY_HMAC_SECRET"),

		MailgunDomain: os.Getenv("MAILGUN_DOMAIN"),
		MailgunAPIKey: os.Getenv("MAILGUN_API_KEY"),
		MailgunSender: os.Getenv("MAILGUN_SENDER"),

		FoodAPIURL:     os.Getenv("FOOD_API_URL"),
		FoodAPITimeout: foodTimeout,
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.DBHost == "" {
		return fmt.Errorf("DBHost is not configured")
	}
	if cfg.DBPort == "" {
		return fmt.Errorf("DBPort is not configured")
	}
	if cfg.DBUser == "" {
		return fmt.Errorf("DBUser is not configured")
	}
	if cfg.DBPassword == "" {
		return fmt.Errorf("DBPassword is not configured")
	}
	if cfg.DBName == "" {
		return fmt.Errorf("DBName is not configured")
	}

	if cfg.ShopifyAPIVersion == "" {
		return fmt.Errorf("ShopifyAPIVersion is not configured")
	}
	if cfg.ShopifyAdminToken == "" {
		return fmt.Errorf("ShopifyAdminToken is not configured")
	}
	if cfg.ShopifyStoreName == "" {
		return fmt.Errorf("ShopifyStoreName is not configured")
	}
	if cfg.ShopifyHMACSecret == "" {
		return fmt.Errorf("ShopifyHMACSecret is not configured")
	}

	if cfg.MailgunDomain == "" {
		return fmt.Errorf("MailgunDomain is not configured")
	}
	if cfg.MailgunAPIKey == "" {
		return fmt.Errorf("MailgunAPIKey is not configured")
	}
	if cfg.MailgunSender == "" {
		return fmt.Errorf("MailgunSender is not configured")
	}

	if cfg.FoodAPIURL == "" {
		return fmt.Errorf("FoodAPIURL is not configured")
	}
	if cfg.FoodAPITimeout <= 0 {
		return fmt.Errorf("FoodAPITimeout must be greater than 0")
	}

	return nil
}
