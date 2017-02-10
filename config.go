package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
)

type Config struct {
	Log struct {
		LogFiles    map[string]string `json:"logFiles"`
		LogzioToken string            `json:"logzioToken"`
	} `json:"log"`
	Discord struct {
		Token                string `json:"token"`
		GuildID              string `json:"guildID"`
		ChannelID            string `json:"channelID"`
		BotAdminRoleID       string `json:"botAdminRoleID"`
		Verbose              bool   `json:"verbose"`
		Debug                bool   `json:"debug"`
		NotificationWarning  int    `json:"notificationWarning"`
		NotificationCritical int    `json:"notificationCritical"`
	} `json:"discord"`
	EVE struct {
		KeyID                 string `json:"keyID"`
		KeyVCode              string `json:"keyvCode"`
		IgnoredStarbases      []int  `json:"ignoredStarbases"`
		MonitorInterval       int    `json:"monitorInterval"`
		FuelWarningThreshold  int    `json:"fuelWarningThreshold"`
		FuelCriticalThreshold int    `json:"fuelCriticalThreshold"`
	} `json:"eve"`
	Redis struct {
		Address  string `json:"address"`
		Password string `json:"password"`
		Database int    `json:"database"`
	} `json:"redis"`
	MySQL struct {
		Address  string `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
	} `json:"mysql"`
}

func parseConfigFile(configFile string) (*Config, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "Config file does not exist")
	}

	file, err := os.Open(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open config file")
	}

	config := &Config{}

	parser := json.NewDecoder(file)
	if err = parser.Decode(config); err != nil {
		return nil, errors.Wrap(err, "Failed to parse config file")
	}

	if len(config.Discord.Token) == 0 || len(config.Discord.GuildID) == 0 || len(config.Discord.ChannelID) == 0 {
		return nil, errors.New("Discord config missing required data")
	}
	if len(config.EVE.KeyID) == 0 || len(config.EVE.KeyVCode) == 0 {
		return nil, errors.New("EVE config missing required data")
	}
	if len(config.Redis.Address) == 0 {
		return nil, errors.New("Redis config missing required data")
	}
	if len(config.MySQL.Address) == 0 || len(config.MySQL.Username) == 0 || len(config.MySQL.Password) == 0 || len(config.MySQL.Database) == 0 {
		return nil, errors.New("MySQL config missing required data")
	}

	return config, nil
}
