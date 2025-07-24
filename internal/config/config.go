package config

import (
	"log"
	"os"
	//"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	//"gopkg.in/yaml.v3"
)

type Config struct {
	Env                     string `yaml:"env"`
	StorageConnectionString string `yaml:"storage_connection_string"`
	HTTPServer              `yaml:"http_server"`
}

type HTTPServer struct {
	Address     string        `yaml:"address"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
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
