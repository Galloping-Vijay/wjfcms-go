package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	Log       LogConfig
	DB        DBConfig
	JWT       JWTConfig
	CORS      CORSConfig
	Upload    UploadConfig
	Runtime   RuntimeConfig
	Redis     RedisConfig
	Mail      MailConfig
	AWS       AWSConfig
	Pusher    PusherConfig
	Baijiahao BaijiahaoConfig
	Tuling    TulingConfig
	QQAI      QQAIConfig
	BaiduSite BaiduSiteConfig
	Wechat    WechatOfficialConfig
	WechatWeb WechatWebConfig
	OAuth     OAuthConfig
}

type AppConfig struct {
	Name         string
	Env          string
	Key          string
	Debug        bool
	Port         string
	URL          string
	ConsoleColor bool
}

type LogConfig struct {
	Channel string
}

type DBConfig struct {
	Host            string
	Port            string
	Database        string
	Username        string
	Password        string
	Prefix          string
	LogSQL          bool
	LogSlowSQL      bool
	LogErrorSQL     bool
	LogLevel        string
	SlowThresholdMS int
}

type JWTConfig struct {
	Secret                string
	ExpiresMinutes        int
	RefreshExpiresMinutes int
}

type CORSConfig struct {
	AllowOrigins []string
}

type UploadConfig struct {
	PublicDir string
	BasePath  string
}

type RuntimeConfig struct {
	BroadcastDriver string
	CacheDriver     string
	QueueConnection string
	SessionDriver   string
	SessionLifetime int
}

type RedisConfig struct {
	Host     string
	Password string
	Port     string
}

type MailConfig struct {
	Driver     string
	Host       string
	Port       string
	Username   string
	Password   string
	Encryption string
	From       string
	To         string
}

type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	DefaultRegion   string
	Bucket          string
}

type PusherConfig struct {
	AppID   string
	AppKey  string
	Secret  string
	Cluster string
}

type BaijiahaoConfig struct {
	AppID          string
	AppToken       string
	AppYouToken    string
	EncodingAESKey string
}

type TulingConfig struct {
	APIKey string
	APIURL string
}

type QQAIConfig struct {
	AppID  string
	AppKey string
	URL    string
}

type WechatOfficialConfig struct {
	AppID  string
	Secret string
	Token  string
	AESKey string
}

type BaiduSiteConfig struct {
	Base string
	API  string
}

type WechatWebConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type OAuthConfig struct {
	Github OAuthProviderConfig
	QQ     OAuthProviderConfig
	Weibo  OAuthProviderConfig
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

func Load(files ...string) Config {
	loadEnv(files...)
	appEnv := env("APP_ENV", "local")
	appKey := env("APP_KEY", "")
	jwtSecret := env("JWT_SECRET", "")
	if jwtSecret == "" || jwtSecret == "change-me" {
		jwtSecret = appKey
	}
	if jwtSecret == "" {
		jwtSecret = "change-me"
	}

	return Config{
		App: AppConfig{
			Name:         env("APP_NAME", "wjfcm-go"),
			Env:          appEnv,
			Key:          appKey,
			Debug:        envBool("APP_DEBUG", !isProductionEnv(appEnv)),
			Port:         env("APP_PORT", "8080"),
			URL:          env("APP_URL", "http://localhost:8080"),
			ConsoleColor: envBool("APP_CONSOLE_COLOR", true),
		},
		Log: LogConfig{
			Channel: env("LOG_CHANNEL", "stack"),
		},
		DB: DBConfig{
			Host:            env("DB_HOST", "127.0.0.1"),
			Port:            env("DB_PORT", "3306"),
			Database:        env("DB_DATABASE", "homestead"),
			Username:        env("DB_USERNAME", "homestead"),
			Password:        env("DB_PASSWORD", "secret"),
			Prefix:          env("DB_PREFIX", "wjf_"),
			LogSQL:          envBool("DB_LOG_SQL", false),
			LogSlowSQL:      envBool("DB_LOG_SLOW_SQL", true),
			LogErrorSQL:     envBool("DB_LOG_ERROR_SQL", true),
			LogLevel:        env("DB_LOG_LEVEL", "info"),
			SlowThresholdMS: envInt("DB_SLOW_THRESHOLD_MS", 200),
		},
		JWT: JWTConfig{
			Secret:                jwtSecret,
			ExpiresMinutes:        envInt("JWT_EXPIRES_MINUTES", 120),
			RefreshExpiresMinutes: envInt("JWT_REFRESH_EXPIRES_MINUTES", 10080),
		},
		CORS: CORSConfig{
			AllowOrigins: envList("CORS_ALLOW_ORIGINS", []string{"http://localhost:5173", "http://127.0.0.1:5173"}),
		},
		Upload: UploadConfig{
			PublicDir: env("PUBLIC_DIR", "../public"),
			BasePath:  env("UPLOAD_BASE_PATH", "uploads"),
		},
		Runtime: RuntimeConfig{
			BroadcastDriver: env("BROADCAST_DRIVER", "log"),
			CacheDriver:     env("CACHE_DRIVER", "file"),
			QueueConnection: env("QUEUE_CONNECTION", "sync"),
			SessionDriver:   env("SESSION_DRIVER", "file"),
			SessionLifetime: envInt("SESSION_LIFETIME", 120),
		},
		Redis: RedisConfig{
			Host:     env("REDIS_HOST", "127.0.0.1"),
			Password: envNullable("REDIS_PASSWORD", ""),
			Port:     env("REDIS_PORT", "6379"),
		},
		Mail: MailConfig{
			Driver:     env("MAIL_DRIVER", "smtp"),
			Host:       env("MAIL_HOST", "smtp.mailtrap.io"),
			Port:       env("MAIL_PORT", "465"),
			Username:   envNullable("MAIL_USERNAME", ""),
			Password:   envNullable("MAIL_PASSWORD", ""),
			Encryption: envNullable("MAIL_ENCRYPTION", ""),
			From:       envNullable("MAIL_FROM_ADDRESS", envNullable("MAIL_USERNAME", "")),
			To:         envNullable("MAIL_TO_ADDRESS", envNullable("MAIL_USERNAME", "")),
		},
		AWS: AWSConfig{
			AccessKeyID:     env("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: env("AWS_SECRET_ACCESS_KEY", ""),
			DefaultRegion:   env("AWS_DEFAULT_REGION", "us-east-1"),
			Bucket:          env("AWS_BUCKET", ""),
		},
		Pusher: PusherConfig{
			AppID:   env("PUSHER_APP_ID", ""),
			AppKey:  env("PUSHER_APP_KEY", ""),
			Secret:  env("PUSHER_APP_SECRET", ""),
			Cluster: env("PUSHER_APP_CLUSTER", "mt1"),
		},
		Baijiahao: BaijiahaoConfig{
			AppID:          env("BAIJIAHAO_APP_ID", ""),
			AppToken:       env("BAIJIAHAO_APP_TOKEN", ""),
			AppYouToken:    env("BAIJIAHAO_APP_YOU_TOKEN", ""),
			EncodingAESKey: env("BAIJIAHAO_APP_Encoding_AESKe", env("BAIJIAHAO_APP_ENCODING_AES_KEY", "")),
		},
		Tuling: TulingConfig{
			APIKey: env("TULING_API_KEY", ""),
			APIURL: env("TULING_API_URL", ""),
		},
		QQAI: QQAIConfig{
			AppID:  env("QQ_AI_APPID", ""),
			AppKey: env("QQ_AI_APPKEY", ""),
			URL:    env("QQ_AI_URL", ""),
		},
		BaiduSite: BaiduSiteConfig{
			Base: env("BAIDU_SITE_BASE", ""),
			API:  env("BAIDU_SITE_API", ""),
		},
		Wechat: WechatOfficialConfig{
			AppID:  env("WECHAT_OFFICIAL_ACCOUNT_APPID", ""),
			Secret: env("WECHAT_OFFICIAL_ACCOUNT_SECRET", ""),
			Token:  env("WECHAT_OFFICIAL_ACCOUNT_TOKEN", ""),
			AESKey: env("WECHAT_OFFICIAL_ACCOUNT_AES_KEY", ""),
		},
		WechatWeb: WechatWebConfig{
			ClientID:     env("WECHATWEB_CLIENT_ID", ""),
			ClientSecret: env("WECHATWEB_CLIENT_SECRET", ""),
			RedirectURI:  env("WECHATWEB_REDIRECT_URI", ""),
		},
		OAuth: OAuthConfig{
			Github: OAuthProviderConfig{
				ClientID:     env("GITHUB_CLIENT_ID", ""),
				ClientSecret: env("GITHUB_CLIENT_SECRET", ""),
				RedirectURI:  env("GITHUB_REDIRECT", ""),
			},
			QQ: OAuthProviderConfig{
				ClientID:     env("QQ_CLIENT_ID", ""),
				ClientSecret: env("QQ_CLIENT_SECRET", ""),
				RedirectURI:  env("QQ_REDIRECT", ""),
			},
			Weibo: OAuthProviderConfig{
				ClientID:     env("WEIBO_CLIENT_ID", ""),
				ClientSecret: env("WEIBO_CLIENT_SECRET", ""),
				RedirectURI:  env("WEIBO_REDIRECT", ""),
			},
		},
	}
}

func (c AppConfig) IsProduction() bool {
	return isProductionEnv(c.Env)
}

func isProductionEnv(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "production" || value == "prod"
}

func loadEnv(files ...string) {
	customFiles := make([]string, 0, len(files))
	for _, file := range files {
		if file = strings.TrimSpace(file); file != "" {
			customFiles = append(customFiles, file)
		}
	}
	if len(customFiles) > 0 {
		_ = godotenv.Load(customFiles...)
		return
	}

	_ = godotenv.Load(".env")
	_ = godotenv.Load(filepath.Clean("../.env"))
	_ = godotenv.Load(filepath.Clean("../../.env"))
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envNullable(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if strings.EqualFold(value, "null") {
		return ""
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1"
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
