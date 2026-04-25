package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	RedisHost    string
	RedisPort    string
	JWTSecret    string
	UploadDir    string
	MaxStorageMB int64
	ChunkSize    int64
	ServerPort   string
}

func Load() *Config {
	maxStorage, _ := strconv.ParseInt(getEnv("MAX_STORAGE_MB", "100"), 10, 64)
	chunkSize, _ := strconv.ParseInt(getEnv("CHUNK_SIZE", "2097152"), 10, 64)

	return &Config{
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "cloud_disk"),
		DBPassword:   getEnv("DB_PASSWORD", "cloud_disk_password"),
		DBName:       getEnv("DB_NAME", "cloud_disk"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		JWTSecret:    getEnv("JWT_SECRET", "your_jwt_secret_key_change_in_production"),
		UploadDir:    getEnv("UPLOAD_DIR", "./uploads"),
		MaxStorageMB: maxStorage,
		ChunkSize:    chunkSize,
		ServerPort:   getEnv("SERVER_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
