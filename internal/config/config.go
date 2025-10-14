package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	DSN        string
	JWTSecret  string
	JWTTTLHrs  int
	Env        string
}

func Load() *Config {
	_ = godotenv.Load()
	ttl, err := strconv.Atoi(getEnv("JWT_TTL_HOURS", "24"))
	if err != nil { ttl = 24 }

	c := &Config{
		Port:      getEnv("PORT", "8080"),
		DSN:       mustEnv("DB_DSN"),
		JWTSecret: mustEnv("JWT_SECRET"),
		JWTTTLHrs: ttl,
		Env:       getEnv("ENV", "dev"),
	}
	log.Printf("config loaded: env=%s port=%s", c.Env, c.Port)
	return c
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" { log.Fatalf("missing env: %s", k) }
	return v
}
