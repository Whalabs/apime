package config

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v10"
)

// Config representa as variáveis principais da aplicação.
type Config struct {
	App      AppConfig
	DB       DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Log      LogConfig
	Storage  StorageConfig
	WhatsApp WhatsAppConfig
}

type StorageConfig struct {
	Driver string `env:"STORAGE_DRIVER" envDefault:"memory"`
}

type AppConfig struct {
	Env     string `env:"APP_ENV" envDefault:"development"`
	Port    string `env:"PORT" envDefault:"8080"`
	BaseURL string `env:"APP_BASE_URL" envDefault:"http://localhost:8080"`
}

type DatabaseConfig struct {
	URL      string `env:"DATABASE_URL"`
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     int    `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"postgres"`
	Password string `env:"DB_PASSWORD" envDefault:"postgres"`
	Name     string `env:"DB_NAME" envDefault:"postgres"`
	SSLMode  string `env:"DB_SSLMODE" envDefault:"disable"`
}

// DSN retorna a string de conexão em formato aceito pelo pgxpool.
func (cfg DatabaseConfig) DSN() string {
	if cfg.URL != "" {
		return cfg.URL
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

type JWTConfig struct {
	Secret   string `env:"JWT_SECRET,required"`
	ExpHours int    `env:"JWT_EXP_HOURS" envDefault:"24"`
}

type LogConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"debug"`
}

type WhatsAppConfig struct {
	SessionKeyEnc string `env:"WHATSAPP_SESSION_KEY_ENC" envDefault:"apime-session-key-change-in-production"`
	SessionDir    string `env:"WHATSAPP_SESSION_DIR" envDefault:"/app/sessions"`
}


// Load carrega as configurações da aplicação.
func Load() Config {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("config: não foi possível carregar variáveis: %v", err)
	}
	return cfg
}
