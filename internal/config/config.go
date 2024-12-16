package config

import (
	"encoding/json"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Storage  StorageConfig
	Worker   WorkerConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Host string
	Port int
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
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
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

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvAsInt("SERVER_PORT", 8080),
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
				Bucket:    getEnv("S3_BUCKET", ""),
				Region:    getEnv("S3_REGION", ""),
				AccessKey: getEnv("S3_ACCESS_KEY", ""),
				SecretKey: getEnv("S3_SECRET_KEY", ""),
			},
		},
		Worker: WorkerConfig{
			Concurrency: getEnvAsInt("WORKER_CONCURRENCY", 5),
			QueueSize:   getEnvAsInt("WORKER_QUEUE_SIZE", 100),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			Username: getEnv("REDIS_USERNAME", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
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

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
