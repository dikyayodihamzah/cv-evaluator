package env

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	envMap = make(map[string]string)
	once   sync.Once
)

func init() {
	loadEnv()
}

func loadEnv() {
	once.Do(func() {
		// Try to load .env file
		file, err := os.Open(".env")
		if err != nil {
			return // If .env doesn't exist, just use system env
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				envMap[key] = value
			}
		}
	})
}

func GetString(key string, defaultValue ...string) string {
	// First check system environment
	if value := os.Getenv(key); value != "" {
		return value
	}

	// Then check .env file
	if value, exists := envMap[key]; exists {
		return value
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}

	// Built-in default values for DeepSeek and other services
	switch key {
	case "LLM_MODEL":
		if GetString("DEEPSEEK_API_KEY") != "" {
			return "deepseek-chat"
		}
		return "gpt-4"
	case "LLM_BASE_URL":
		if GetString("DEEPSEEK_API_KEY") != "" {
			return "https://api.deepseek.com"
		}
		return "https://api.openai.com/v1"
	case "EMBEDDING_MODEL":
		return "text-embedding-ada-002"
	case "ENVIRONMENT":
		return "DEVELOPMENT"
	case "PORT":
		return "8080"
	case "MINIO_BUCKET":
		return "cv-evaluator"
	case "MINIO_USE_SSL":
		return "false"
	default:
		return ""
	}
}

func GetBool(key string, defaultValue ...bool) bool {
	var value bool
	if len(defaultValue) > 0 {
		value = defaultValue[0]
	}

	valueStr := GetString(key, strconv.FormatBool(value))
	return strings.ToLower(valueStr) == "true"
}
