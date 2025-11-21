package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 應用程式配置結構
type Config struct {
	// 服務器配置
	Port        string
	Host        string
	Environment string
	FrontendURL string

	// 資料庫配置
	Database DatabaseConfig
	Redis    RedisConfig

	// JWT 配置
	JWTSecret string

	// CORS 配置
	CORSOrigins []string

	// WebSocket 配置
	WebSocket WebSocketConfig

	// 遊戲配置
	Game GameConfig

	// 日誌配置
	Log LogConfig
}



// DatabaseConfig PostgreSQL 資料庫配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	URL      string // 直接使用連接字串
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	URL      string // 直接使用連接字串
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	ReadBufferSize  int
	WriteBufferSize int
	MaxMessageSize  int64
}

// GameConfig 遊戲相關配置
type GameConfig struct {
	MaxPlayersPerRoom      int
	RoomIDLength           int
	QuestionTimeLimit      int
	DefaultTotalQuestions  int
}

// LogConfig 日誌配置
type LogConfig struct {
	Level  string
	Format string
}

// Load 載入配置
func Load() *Config {
	frontendURL := strings.TrimSuffix(getEnv("FRONTEND_URL", "http://localhost:5173"), "/")
	corsOrigins := getEnvAsSlice("CORS_ORIGINS", []string{
		"http://localhost:5173",
		"http://localhost:3000",
	})

	if frontendURL != "" {
		matched := false
		for _, origin := range corsOrigins {
			if strings.TrimSuffix(origin, "/") == frontendURL {
				matched = true
				break
			}
		}
		if !matched {
			corsOrigins = append(corsOrigins, frontendURL)
		}
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		Host:        getEnv("HOST", "localhost"),
		Environment: getEnv("ENV", "development"),
		FrontendURL: frontendURL,

		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			Name:     getEnv("DB_NAME", "kahoot_game"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			URL:      getEnv("DATABASE_URL", ""),
		},

		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			URL:      getEnv("REDIS_URL", ""),
		},

		JWTSecret: getEnv("JWT_SECRET", "your-super-secret-jwt-key"),

		CORSOrigins: corsOrigins,

		WebSocket: WebSocketConfig{
			ReadBufferSize:  getEnvAsInt("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize: getEnvAsInt("WS_WRITE_BUFFER_SIZE", 1024),
			MaxMessageSize:  int64(getEnvAsInt("WS_MAX_MESSAGE_SIZE", 512)),
		},

		Game: GameConfig{
			MaxPlayersPerRoom:     getEnvAsInt("MAX_PLAYERS_PER_ROOM", 20),
			RoomIDLength:          getEnvAsInt("ROOM_ID_LENGTH", 6),
			QuestionTimeLimit:     getEnvAsInt("QUESTION_TIME_LIMIT", 30),
			DefaultTotalQuestions: getEnvAsInt("DEFAULT_TOTAL_QUESTIONS", 10),
		},

		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}
}

// getEnv 獲取環境變數，如果不存在則返回預設值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt 獲取環境變數並轉換為整數
func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// getEnvAsSlice 獲取環境變數並轉換為字串切片
func getEnvAsSlice(key string, defaultValue []string) []string {
	if valueStr := os.Getenv(key); valueStr != "" {
		return strings.Split(valueStr, ",")
	}
	return defaultValue
}

// GetDatabaseDSN 獲取資料庫連線字串
func (c *Config) GetDatabaseDSN() string {
	if c.Database.URL != "" {
		return c.Database.URL
	}
	return "host=" + c.Database.Host +
		" port=" + c.Database.Port +
		" user=" + c.Database.User +
		" password=" + c.Database.Password +
		" dbname=" + c.Database.Name +
		" sslmode=" + c.Database.SSLMode
}

// GetRedisAddr 獲取 Redis 地址
// 注意：如果使用 URL，此方法可能不適用，建議直接在 redis.go 中處理
func (c *Config) GetRedisAddr() string {
	return c.Redis.Host + ":" + c.Redis.Port
}
