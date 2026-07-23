package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server   ServerConfig
	MySQL    MySQLConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Email    EmailConfig
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
}

type ServerConfig struct {
	Host            string        `yaml:"host" env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port            string        `yaml:"port" env:"SERVER_PORT" env-default:"8080"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type MySQLConfig struct {
	Host            string        `yaml:"host" env:"MYSQL_HOST" env-default:"127.0.0.1"`
	Port            string        `yaml:"port" env:"MYSQL_PORT" env-default:"3306"`
	User            string        `yaml:"user" env:"MYSQL_USER" env-required:"true"`
	Password        string        `yaml:"password" env:"MYSQL_PASSWORD" env-required:"true"`
	Database        string        `yaml:"database" env:"MYSQL_DATABASE" env-required:"true"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"MYSQL_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"MYSQL_MAX_IDLE_CONNS" env-default:"10"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"MYSQL_CONN_MAX_LIFETIME" env-default:"30m"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr" env:"REDIS_ADDR" env-default:"127.0.0.1:6379"`
	Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
	DB       int    `yaml:"db" env:"REDIS_DB" env-default:"0"`
}

type JWTConfig struct {
	Secret string        `yaml:"secret" env:"JWT_SECRET" env-required:"true"`
	TTL    time.Duration `yaml:"ttl" env:"JWT_TTL" env-default:"24h"`
}

type EmailConfig struct {
	FailureRate        float64       `yaml:"failure_rate" env:"EMAIL_FAILURE_RATE" env-default:"0.2"`
	Latency            time.Duration `yaml:"latency" env:"EMAIL_LATENCY" env-default:"150ms"`
	BreakerMaxRequests uint32        `yaml:"breaker_max_requests" env:"EMAIL_BREAKER_MAX_REQUESTS" env-default:"3"`
	BreakerInterval    time.Duration `yaml:"breaker_interval" env:"EMAIL_BREAKER_INTERVAL" env-default:"30s"`
	BreakerTimeout     time.Duration `yaml:"breaker_timeout" env:"EMAIL_BREAKER_TIMEOUT" env-default:"15s"`
}

// Load читает файл по пути CONFIG_PATH (по умолчанию config/config.yaml),
// если он существует, иначе использует только переменные окружения.
func Load() (*Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config/config.yaml"
	}

	var cfg Config
	if _, err := os.Stat(path); err == nil {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
