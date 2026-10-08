package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort           string
	DBHost            string
	DBPort            string
	DBName            string
	DBUser            string
	DBPassword        string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	HTTPReadTimeout   time.Duration
	HTTPWriteTimeout  time.Duration
	HTTPIdleTimeout   time.Duration
	Timezone          string
}

func Load() (Config, error) {
	c := Config{
		AppPort:           env("APP_PORT", "8080"),
		DBHost:            os.Getenv("DB_HOST"),
		DBPort:            env("DB_PORT", "3306"),
		DBName:            os.Getenv("DB_NAME"),
		DBUser:            os.Getenv("DB_USER"),
		DBPassword:        os.Getenv("DB_PASSWORD"),
		DBMaxOpenConns:    intEnv("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:    intEnv("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: durationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		HTTPReadTimeout:   durationEnv("HTTP_READ_TIMEOUT", 30*time.Second),
		HTTPWriteTimeout:  durationEnv("HTTP_WRITE_TIMEOUT", 60*time.Second),
		HTTPIdleTimeout:   durationEnv("HTTP_IDLE_TIMEOUT", 120*time.Second),
		Timezone:          env("TZ", "Asia/Jakarta"),
	}
	if c.DBHost == "" || c.DBName == "" || c.DBUser == "" {
		return Config{}, fmt.Errorf("DB_HOST, DB_NAME, and DB_USER must be configured")
	}
	return c, nil
}

func (c Config) ListenAddr() string { return ":" + c.AppPort }

func (c Config) DatabaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Asia%%2FJakarta", c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(env(key, ""))
	if err != nil {
		return fallback
	}
	return value
}
