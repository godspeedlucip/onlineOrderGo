package domain

type Config struct {
	App         AppConfig
	HTTP        HTTPConfig
	DB          DBConfig
	Redis       RedisConfig
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

type DBConfig struct {
	Driver string
	DSN    string
}

type RedisConfig struct {
	Addr      string
	Password  string
	DB        int
	KeyPrefix string
}

type LogConfig struct {
	Level string
}

type IdempotencyConfig struct {
	Enabled   bool
	TTLSecond int
}
