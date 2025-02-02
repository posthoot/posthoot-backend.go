package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Worker   WorkerConfig
	Redis    RedisConfig
	S3       S3Config
	Crypto   CryptoConfig
	SMTP     SMTPConfig
}

type CryptoConfig struct {
	PrivateKey string
}

type ServerConfig struct {
	Host      string
	Port      int
	PublicURL string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret string
}

type StorageConfig struct {
	Provider string // local, s3, etc.
	BasePath string
	S3       S3Config
}

type S3Config struct {
	BucketName string `env:"S3_BUCKET_NAME" required:"true"`
	Endpoint   string `env:"S3_ENDPOINT"`
	Region     string `env:"S3_REGION" required:"true"`
	AccessKey  string `env:"S3_ACCESS_KEY" required:"true"`
	SecretKey  string `env:"S3_SECRET_KEY" required:"true"`
}

type SMTPConfig struct {
	Host      string `env:"SMTP_HOST" required:"true"`
	Port      int    `env:"SMTP_PORT" required:"true"`
	User      string `env:"SMTP_USER" required:"true"`
	Password  string `env:"SMTP_PASSWORD" required:"true"`
	FromEmail string `env:"SMTP_FROM_EMAIL" required:"true"`
	Provider  string `env:"SMTP_PROVIDER" required:"true"`
}

type WorkerConfig struct {
	Concurrency int
	QueueSize   int
}

type RedisConfig struct {
	Addr     string
	Password string
	Username string
	DB       int
}

var (
	config *Config
	once   sync.Once
)

// GetConfig returns the singleton config instance
func GetConfig() *Config {
	once.Do(func() {
		config = &Config{}
		config.JWT.Secret = os.Getenv("JWT_SECRET")
		config.Server.PublicURL = os.Getenv("PUBLIC_URL")
	})
	return config
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:      getEnv("SERVER_HOST", "localhost"),
			Port:      getEnvAsInt("SERVER_PORT", 8080),
			PublicURL: getEnv("PUBLIC_URL", "http://localhost:8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvAsInt("POSTGRES_PORT", 5432),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", ""),
			Name:     getEnv("POSTGRES_DB", "kori"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-secret-key"),
		},
		Storage: StorageConfig{
			Provider: getEnv("STORAGE_PROVIDER", "local"),
			BasePath: getEnv("STORAGE_BASE_PATH", "./storage"),
			S3: S3Config{
				BucketName: getEnv("S3_BUCKET_NAME", ""),
				Endpoint:   getEnv("S3_ENDPOINT", ""),
				Region:     getEnv("S3_REGION", ""),
				AccessKey:  getEnv("S3_ACCESS_KEY", ""),
				SecretKey:  getEnv("S3_SECRET_KEY", ""),
			},
		},
		Worker: WorkerConfig{
			Concurrency: getEnvAsInt("WORKER_CONCURRENCY", 5),
			QueueSize:   getEnvAsInt("WORKER_QUEUE_SIZE", 100),
		},
		Redis: RedisConfig{
			Addr:     fmt.Sprintf("%s:%d", getEnv("REDIS_HOST", "localhost"), getEnvAsInt("REDIS_PORT", 6379)),
			Password: getEnv("REDIS_PASSWORD", ""),
			Username: getEnv("REDIS_USERNAME", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Crypto: CryptoConfig{
			PrivateKey: getEnv("PRIVATE_KEY", ""),
		},
		SMTP: SMTPConfig{
			Host:      getEnv("SMTP_HOST", ""),
			Port:      getEnvAsInt("SMTP_PORT", 587),
			User:      getEnv("SMTP_USER", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("SMTP_FROM_EMAIL", ""),
			Provider:  getEnv("SMTP_PROVIDER", "CUSTOM"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
