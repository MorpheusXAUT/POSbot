package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
	"github.com/morpheusxaut/eveapi"
	"github.com/pkg/errors"
	"os"
	"time"
)

const (
	UserAgent string = "POSbot v" + Version + " - github.com/MorpheusXAUT/POSbot"
)

type Bot struct {
	discord *discordgo.Session
	eve     eveapi.API
	redis   *redis.Pool

	config *BotConfig
}

type BotConfig struct {
	Discord struct {
		Token     string `json:"token"`
		GuildID   string `json:"guildID"`
		ChannelID string `json:"channelID"`
		Verbose   bool   `json:"verbose"`
		Debug     bool   `json:"debug"`
	} `json:"discord"`
	EVE struct {
		KeyID            string   `json:"keyID"`
		KeyVCode         string   `json:"keyvCode"`
		IgnoredStarbases []string `json:"ignoredStarbases"`
	} `json:"eve"`
	Redis struct {
		Address  string `json:"address"`
		Password string `json:"password"`
		Database int    `json:"database"`
	} `json:"redis"`
}

func NewBot(config *BotConfig) (*Bot, error) {
	bot := &Bot{
		config: config,
	}

	var err error

	log.Debug("Initialising Discord connection")
	bot.discord, err = discordgo.New(fmt.Sprintf("Bot %s", bot.config.Discord.Token))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create Discord session")
	}

	bot.discord.AddHandler(bot.onDiscordReady)
	bot.discord.AddHandler(bot.onDiscordGuildCreate)
	bot.discord.AddHandler(bot.onDiscordMessageCreate)

	err = bot.discord.Open()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open Discord session")
	}

	log.Debug("Initialising EVE connection")
	bot.eve = eveapi.API{
		Server: eveapi.Tranquility,
		APIKey: eveapi.Key{
			ID:    bot.config.EVE.KeyID,
			VCode: bot.config.EVE.KeyVCode,
		},
		UserAgent: UserAgent,
		Timeout:   60 * time.Second,
		Debug:     false,
	}

	_, err = bot.eve.ServerStatus()
	if err != nil {
		bot.discord.Close()
		return nil, errors.Wrap(err, "Failed to query EVE server status")
	}

	log.Debug("Initialising Redis connection")
	redisOptions := make([]redis.DialOption, 0)
	if len(bot.config.Redis.Password) > 0 {
		redisOptions = append(redisOptions, redis.DialPassword(bot.config.Redis.Password))
	}
	if bot.config.Redis.Database >= 0 {
		redisOptions = append(redisOptions, redis.DialDatabase(bot.config.Redis.Database))
	}

	bot.redis = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", bot.config.Redis.Address, redisOptions...)
		},
	}

	r := bot.redis.Get()
	defer r.Close()

	_, err = r.Do("PING")
	if err != nil {
		bot.discord.Close()
		return nil, errors.Wrap(err, "Failed to ping Redis server")
	}

	return bot, nil
}

func NewBotFromConfigFile(configFile string) (*Bot, error) {
	config, err := parseConfigFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create bot from config file")
	}

	return NewBot(config)
}

func (b *Bot) Shutdown() {
	log.Debug("Clean bot shutdown initiated")

	if b.config.Discord.Debug {
		b.discord.ChannelMessageSend(b.config.Discord.ChannelID, ":robot: POSbot shutting down :skull_crossbones:")
	}

	b.discord.Close()
	b.redis.Close()

	log.Debug("Clean bot shutdown completed")
}

func parseConfigFile(configFile string) (*BotConfig, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "Config file does not exist")
	}

	file, err := os.Open(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open config file")
	}

	config := &BotConfig{}

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

	return config, nil
}
