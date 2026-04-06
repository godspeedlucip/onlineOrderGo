package domain

type Config struct {
	App         AppConfig
	HTTP        HTTPConfig
	Log         LogConfig
	Idempotency IdempotencyConfig
}

type AppConfig struct {
	Name string
	Env  string
}

type HTTPConfig struct {
	Addr string
}

type LogConfig struct {
	Level string
}

type IdempotencyConfig struct {
	Enabled   bool
	TTLSecond int
}