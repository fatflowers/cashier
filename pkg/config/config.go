package config

import (
	"context"
	"fmt"
	"github.com/fatflowers/cashier/pkg/types"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DBConfig struct {
	DSN string `mapstructure:"dsn"`
}

type Env string

const (
	EnvDev  Env = "dev"
	EnvProd Env = "prod"
)

type Config struct {
	Env          Env                  `mapstructure:"env"`
	Server       ServerConfig         `mapstructure:"server"`
	Database     DBConfig             `mapstructure:"database"`
	PaymentItems []*types.PaymentItem `mapstructure:"payment_items"`
	AppleIAP     AppleIAPConfig       `mapstructure:"apple_iap"`
	MetricsAddr  string               `mapstructure:"metrics_addr"`
}

type AppleIAPConfig struct {
	KeyID        string `mapstructure:"key_id"`
	KeyContent   string `mapstructure:"key_content"`
	BundleID     string `mapstructure:"bundle_id"`
	Issuer       string `mapstructure:"issuer"`
	SharedSecret string `mapstructure:"shared_secret"`
	IsProd       bool   `mapstructure:"is_prod"`
}

func (c *Config) GetPaymentItemByID(id string) *types.PaymentItem {
	for _, item := range c.PaymentItems {
		if item.ID == id {
			return item
		}
	}
	return nil
}

func (c *Config) GetPaymentItemByProviderItemID(ctx context.Context, providerID types.PaymentProvider, providerItemID string) (*types.PaymentItem, error) {

	for _, item := range c.PaymentItems {
		if item.ProviderID == providerID && item.ProviderItemID == providerItemID {
			return item, nil
		}
	}
	return nil, fmt.Errorf("payment item not found")
}

func New() (*Config, error) {
	v := viper.New()
	// Allow overriding config file via env:
	// - APP_CONFIG_FILE: absolute or relative file path (e.g., /etc/app/prod.yaml)
	// - APP_CONFIG_NAME: config base name without extension (default: "config")
	if file := os.Getenv("APP_CONFIG_FILE"); file != "" {
		v.SetConfigFile(file)
	} else {
		cfgName := os.Getenv("APP_CONFIG_NAME")
		if cfgName == "" {
			cfgName = "config"
		}
		v.SetConfigName(cfgName)
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("env", "dev")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8888)
	v.SetDefault("database.dsn", "postgres://postgres:postgres@localhost:5432/appdb?sslmode=disable")
	v.SetDefault("metrics_addr", ":90")

	if err := v.ReadInConfig(); err != nil {
		_ = err
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &c, nil
}

var Module = fx.Options(
	fx.Provide(New),
)
