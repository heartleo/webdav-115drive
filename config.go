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
	UID  string `yaml:"uid"`
	CID  string `yaml:"cid"`
	SEID string `yaml:"seid"`
	KID  string `yaml:"kid"`
	Rate int    `yaml:"rate"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Drive  DriveConfig  `yaml:"drive"`
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

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err = viper.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return nil, err
		}
	}

	conf := &Config{}

	if err = viper.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
