package main

import (
	"errors"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Addr   string `yaml:"addr"`
	Port   int    `yaml:"port"`
	Prefix string `yaml:"prefix"`
	Debug  bool   `yaml:"debug"`
	User   string `yaml:"user"`
	Pwd    string `yaml:"pwd"`
}

type DriveConfig struct {
	UID         string `yaml:"uid"`
	CID         string `yaml:"cid"`
	SEID        string `yaml:"seid"`
	KID         string `yaml:"kid"`
	Rate        int    `yaml:"rate"`
	CacheExpire int    `yaml:"cache_expire"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Drive  DriveConfig  `yaml:"drive"`
}

func bindEnvs() {
	_ = viper.BindEnv("server.addr")
	_ = viper.BindEnv("server.port")
	_ = viper.BindEnv("server.prefix")
	_ = viper.BindEnv("server.debug")
	_ = viper.BindEnv("server.user")
	_ = viper.BindEnv("server.pwd")

	_ = viper.BindEnv("drive.uid")
	_ = viper.BindEnv("drive.cid")
	_ = viper.BindEnv("drive.seid")
	_ = viper.BindEnv("drive.kid")
	_ = viper.BindEnv("drive.rate")
	_ = viper.BindEnv("drive.cache_expire")
}

func LoadConfig(configPath string) (*Config, error) {
	var err error

	if configPath == "" {
		configPath, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	if _, err = os.Stat(path.Join(configPath, ".env")); err == nil {
		if err = godotenv.Load(); err != nil {
			return nil, err
		}
		slog.Debug(".env loaded")
	}

	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.SetDefault("server.addr", "0.0.0.0")
	viper.SetDefault("server.port", 8090)
	viper.SetDefault("server.prefix", "/dav")
	viper.SetDefault("server.debug", false)
	viper.SetDefault("server.user", "root")
	viper.SetDefault("server.pwd", "123456")
	viper.SetDefault("drive.rate", 3)
	viper.SetDefault("drive.cache_expire", 1)

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	bindEnvs()

	if err = viper.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return nil, err
		}
	}

	conf := &Config{}

	if err := viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
