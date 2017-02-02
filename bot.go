package main

import (
	"encoding/json"
	"fmt"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/MorpheusXAUT/evesi"
	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gregjones/httpcache"
	httpredis "github.com/gregjones/httpcache/redis"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"time"
)

const (
	UserAgent string = "POSbot v" + Version + " - github.com/MorpheusXAUT/POSbot"
)

var (
	mysqlRequiredTableNames []string = []string{"mapDenormalize", "invTypes"}
)

type Bot struct {
	discord *discordgo.Session
	esi     *evesi.APIClient
	eve     eveapi.API
	http    *http.Client
	mysql   *sqlx.DB
	redis   *redis.Pool

	config    *BotConfig
	startTime time.Time
}

type BotConfig struct {
	Discord struct {
		Token          string `json:"token"`
		GuildID        string `json:"guildID"`
		ChannelID      string `json:"channelID"`
		BotAdminRoleID string `json:"botAdminRoleID"`
		Verbose        bool   `json:"verbose"`
		Debug          bool   `json:"debug"`
	} `json:"discord"`
	EVE struct {
		KeyID            string `json:"keyID"`
		KeyVCode         string `json:"keyvCode"`
		IgnoredStarbases []int  `json:"ignoredStarbases"`
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

func NewBot(config *BotConfig) (*Bot, error) {
	bot := &Bot{
		config:    config,
		startTime: time.Now().UTC(),
	}

	var err error

	log.Debug("Initialising Redis connection")
	redisOptions := make([]redis.DialOption, 0)
	if len(bot.config.Redis.Password) > 0 {
		redisOptions = append(redisOptions, redis.DialPassword(bot.config.Redis.Password))
	}
	if bot.config.Redis.Database >= 0 {
		redisOptions = append(redisOptions, redis.DialDatabase(bot.config.Redis.Database))
	}

	bot.redis = &redis.Pool{
		MaxIdle:     50,
		MaxActive:   0,
		Wait:        false,
		IdleTimeout: 90 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", bot.config.Redis.Address, redisOptions...)
		},
	}

	r := bot.redis.Get()

	_, err = r.Do("PING")
	r.Close()
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to ping Redis server")
	}

	log.Debug("Creating httpcache client")

	transport := httpcache.NewTransport(httpredis.NewWithClient(bot.redis.Get()))
	bot.http = &http.Client{
		Transport: transport,
		Timeout:   time.Second * 90,
	}

	log.Debug("Initialising ESI connection")
	bot.esi = evesi.NewAPIClient(bot.http, UserAgent)

	log.Debug("Initialising EVE connection")
	bot.eve = eveapi.API{
		Server: eveapi.Tranquility,
		APIKey: eveapi.Key{
			ID:    bot.config.EVE.KeyID,
			VCode: bot.config.EVE.KeyVCode,
		},
		UserAgent: UserAgent,
		Client:    bot.http,
		Debug:     false,
	}

	_, err = bot.eve.ServerStatus()
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to query EVE server status")
	}

	log.Debug("Initialising MySQL connection")
	bot.mysql, err = sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", bot.config.MySQL.Username, bot.config.MySQL.Password, bot.config.MySQL.Address, bot.config.MySQL.Database))
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to open connection to MySQL server")
	}

	err = bot.mysql.Ping()
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to ping MySQL server")
	}

	query, args, err := sqlx.In("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name IN (?)", bot.config.MySQL.Database, mysqlRequiredTableNames)
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to prepare required MySQL tables check")
	}

	var tableCount int
	err = bot.mysql.Get(&tableCount, query, args...)
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to check required MySQL tables")
	}
	if tableCount != len(mysqlRequiredTableNames) {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Missing required MySQL tables")
	}

	log.Debug("Initialising Discord connection")
	bot.discord, err = discordgo.New(fmt.Sprintf("Bot %s", bot.config.Discord.Token))
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to create Discord session")
	}

	bot.discord.AddHandler(bot.onDiscordReady)
	bot.discord.AddHandler(bot.onDiscordGuildCreate)
	bot.discord.AddHandler(bot.onDiscordMessageCreate)

	err = bot.discord.Open()
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to open Discord session")
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
	b.mysql.Close()

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
	if len(config.MySQL.Address) == 0 || len(config.MySQL.Username) == 0 || len(config.MySQL.Password) == 0 || len(config.MySQL.Database) == 0 {
		return nil, errors.New("MySQL config missing required data")
	}

	return config, nil
}
