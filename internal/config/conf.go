package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"os"
)

type Config struct {
	ClientId        string `envconfig:"CLIENT_ID"`
	ClientSecret    string `envconfig:"CLIENT_SECRET"`
	RedirectUri     string `envconfig:"REDIRECT_URI"`
	ParseLinkUpwork string `envconfig:"PARSE_LINK_UPWORK"`
	FiltersStr      string `envconfig:"FILTERS_STR"`
}

// Init populates Config struct with values from config file
// located at filepath and environment variables.
func Init() (*Config, error) {
	var cfg Config

	readEnv(&cfg)
	return &cfg, nil
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func readEnv(cfg *Config) {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	err = envconfig.Process("", cfg)
	if err != nil {
		processError(err)
	}
}
