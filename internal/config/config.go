package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Reddit struct {
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
		Username     string `mapstructure:"username"`
		Password     string `mapstructure:"password"`
	} `mapstructure:"reddit"`

	Kafka struct {
		Brokers []string `mapstructure:"brokers"`
		Topic   string   `mapstructure:"topic"`
		GroupID string   `mapstructure:"group_id"`
	} `mapstructure:"kafka"`

	Postgres struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"postgres"`

	API struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"api"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
