package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"os"
)

type Config struct {
	ClientId            string `envconfig:"CLIENT_ID"`
	ClientSecret        string `envconfig:"CLIENT_SECRET"`
	RedirectUri         string `envconfig:"REDIRECT_URI"`
	ParseLinkUpwork     string `envconfig:"PARSE_LINK_UPWORK"`
	FiltersStr          string `envconfig:"FILTERS_STR"`
	ExcludedFiltersStr  string `envconfig:"EXCLUDED_FILTERS_STR"`
	AmoCrmEndPoint      string `envconfig:"AMOCRM_ENDPOINT"`
	ParsePerMinute      string `envconfig:"PARSE_PER_MINUTE"`
	NotionSecret        string `envconfig:"NOTION_SECRET"`
	AirTableSecret      string `envconfig:"AIRTABLE_SECRET"`
	AirTableDatabase    string `envconfig:"AIRTABLE_DATABASE"`
	AirTableTable       string `envconfig:"AIRTABLE_TABLE"`
	AirTableTableUpwork string `envconfig:"AIRTABLE_TABLE_UPWORK"`
	AirTableTableFL     string `envconfig:"AIRTABLE_TABLE_FL"`
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
