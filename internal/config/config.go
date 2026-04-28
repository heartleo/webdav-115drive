package config

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
	Host     string `yaml:"host"      mapstructure:"host"`
	Port     int    `yaml:"port"      mapstructure:"port"`
	Path     string `yaml:"path"      mapstructure:"path"`
	User     string `yaml:"user"      mapstructure:"user"`
	Password string `yaml:"password"  mapstructure:"password"`
	LogLevel string `yaml:"log_level" mapstructure:"log_level"`
}

type DriveConfig struct {
	UID         string `yaml:"uid"          mapstructure:"uid"`
	CID         string `yaml:"cid"          mapstructure:"cid"`
	SEID        string `yaml:"seid"         mapstructure:"seid"`
	KID         string `yaml:"kid"          mapstructure:"kid"`
	Rate        int    `yaml:"rate"         mapstructure:"rate"`
	CacheExpire int    `yaml:"cache_expire" mapstructure:"cache_expire"`
}

type Config struct {
	Server ServerConfig `yaml:"server" mapstructure:"server"`
	Drive  DriveConfig  `yaml:"drive"  mapstructure:"drive"`
}

func Load(configPath string) (*Config, error) {
	var err error

	if configPath == "" {
		configPath, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	envFile := path.Join(configPath, ".env")
	if _, err = os.Stat(envFile); err == nil {
		if err = godotenv.Load(envFile); err != nil {
			return nil, err
		}
		slog.Debug("loaded .env", slog.String("path", configPath))
	}

	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8090)
	v.SetDefault("server.path", "/dav")
	v.SetDefault("server.user", "user")
	v.SetDefault("server.password", "password")
	v.SetDefault("server.log_level", "info")
	v.SetDefault("drive.uid", "")
	v.SetDefault("drive.cid", "")
	v.SetDefault("drive.seid", "")
	v.SetDefault("drive.kid", "")
	v.SetDefault("drive.rate", 3)
	v.SetDefault("drive.cache_expire", 1)

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err = v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, err
		}
	}

	conf := &Config{}
	if err := v.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
