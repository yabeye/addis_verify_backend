package env

import "os"

// GetString : Gets the value from env
func GetString(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return fallback
}
