package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env                     string `yaml:"env"`
	StorageConnectionString string `yaml:"storage_connection_string"`
	RedisConnection         `yaml:"redis_connection"`
	HTTPServer              `yaml:"http_server"`
	JWTToken                `yaml:"jwttoken"`
}

type HTTPServer struct {
	AddressHttp string        `yaml:"address"`
	TimeoutHttp time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

type RedisConnection struct {
	AddressRedis string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	User         string        `yaml:"user"`
	DB           int           `yaml:"db"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	TimeoutRedis time.Duration `yaml:"timeout"`
}

type JWTToken struct {
	JWTSecretKey string        `yaml:"jwt_secret_key"`
	TokenTTL     time.Duration `yaml:"token_ttl"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("file: %s - does not exist", configPath)
	}
	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}
	return &cfg

}

func (c *Config) String() string {
	return fmt.Sprintf(
		"Env: %s\n"+
			"StorageConnectionString: %s\n"+
			"RedisConnection:\n"+
			"  Addr: %s\n"+
			"  Password: %s\n"+
			"  User: %s\n"+
			"  DB: %d\n"+
			"  MaxRetries: %d\n"+
			"  DialTimeout: %s\n"+
			"  Timeout: %s\n"+
			"HTTPServer:\n"+
			"  Address: %s\n"+
			"  Timeout: %s\n"+
			"  IdleTimeout: %s\n"+
			"JWTToken:\n"+
			"  JWTSecretKey: %s\n"+
			"  TokenTTL: %s\n",
		c.Env,
		c.StorageConnectionString,
		c.AddressRedis,
		c.Password,
		c.User,
		c.DB,
		c.MaxRetries,
		c.DialTimeout,
		c.TimeoutRedis,
		c.AddressHttp,
		c.TimeoutHttp,
		c.IdleTimeout,
		c.JWTSecretKey,
		c.TokenTTL,
	)
}
